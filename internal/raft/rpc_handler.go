package raft

import (
	"context"
	"log"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/internal/snapshot"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

// HandleRequestVote processes an incoming RequestVote RPC.
func (n *Node) HandleRequestVote(
	_ context.Context,
	req *pb.RequestVoteRequest,
) (*pb.RequestVoteResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Reject candidates from an older term.
	if req.Term < n.persistent.CurrentTerm {
		return &pb.RequestVoteResponse{
			Term:        n.persistent.CurrentTerm,
			VoteGranted: false,
		}, nil
	}

	// A newer term always causes this node to become a follower.
	if req.Term > n.persistent.CurrentTerm {
		n.becomeFollowerLocked(req.Term)
	}

	lastLogIndex, lastLogTerm, err := n.lastLogInfoLocked()
	if err != nil {
		return nil, err
	}

	candidateLogIsUpToDate :=
		req.LastLogTerm > lastLogTerm ||
			(req.LastLogTerm == lastLogTerm &&
				req.LastLogIndex >= lastLogIndex)

	canVote :=
		n.persistent.VotedFor == "" ||
			n.persistent.VotedFor == types.NodeID(req.CandidateId)

	voteGranted := canVote && candidateLogIsUpToDate

	if voteGranted {
		n.metrics.VotesGranted++
	} else {
		n.metrics.VotesRejected++
	}

	if voteGranted {
		n.persistent.VotedFor = types.NodeID(req.CandidateId)
		if err := n.persistLocked(); err != nil {
			return nil, err
		}

		n.electionTimer.reset()
	}

	return &pb.RequestVoteResponse{
		Term:        n.persistent.CurrentTerm,
		VoteGranted: voteGranted,
	}, nil
}

// HandleAppendEntries processes heartbeats and replicated log entries.
func (n *Node) HandleAppendEntries(
	_ context.Context,
	req *pb.AppendEntriesRequest,
) (*pb.AppendEntriesResponse, error) {

	n.mu.Lock()
	n.metrics.AppendEntriesRecv++

	// Reject stale leaders.
	if req.Term < n.persistent.CurrentTerm {

		response := &pb.AppendEntriesResponse{
			Term:    n.persistent.CurrentTerm,
			Success: false,
		}

		n.mu.Unlock()

		return response, nil
	}

	defer n.electionTimer.reset()

	// Update term / become follower.
	if req.Term > n.persistent.CurrentTerm {
		n.becomeFollowerLocked(req.Term)
	} else if n.role != Follower {
		n.role = Follower
	}
	n.leaderID = types.NodeID(req.LeaderId)

	// Verify PrevLogIndex exists.
	if req.PrevLogIndex > 0 {

		entry, ok, err := n.wal.EntryAt(req.PrevLogIndex)
		if err != nil {
			n.mu.Unlock()
			return nil, err
		}

		if !ok {

			response := &pb.AppendEntriesResponse{
				Term:    n.persistent.CurrentTerm,
				Success: false,
			}

			n.mu.Unlock()

			return response, nil
		}

		// Verify PrevLogTerm.
		if uint64(entry.Term) != req.PrevLogTerm {

			response := &pb.AppendEntriesResponse{
				Term:    n.persistent.CurrentTerm,
				Success: false,
			}

			n.mu.Unlock()

			return response, nil
		}
	}
	if len(req.Entries) > 0 {

		entries := protobufEntriesToLogEntries(req.Entries)

		if err := n.resolveConflicts(entries); err != nil {
			n.mu.Unlock()
			return nil, err
		}
	}

	if req.LeaderCommit > n.volatile.CommitIndex {
		lastEntry, ok, err := n.wal.LastEntry()
		if err != nil {
			n.mu.Unlock()
			return nil, err
		}
		if ok {
			lastIndex := uint64(lastEntry.Index)
			if req.LeaderCommit < lastIndex {
				n.volatile.CommitIndex = req.LeaderCommit
			} else {
				n.volatile.CommitIndex = lastIndex
			}
			if err := n.applyCommittedEntries(); err != nil {
				n.mu.Unlock()
				return nil, err
			}
		}
	}

	response := &pb.AppendEntriesResponse{
		Term:    n.persistent.CurrentTerm,
		Success: true,
	}

	n.mu.Unlock()

	return response, nil
}

// becomeFollowerLocked transitions the node to follower state.
// The caller must hold n.mu.
func (n *Node) becomeFollowerLocked(term uint64) {
	wasLeader := n.role == Leader
	wasCandidate := n.role == Candidate

	if n.heartbeat != nil {
		n.heartbeat.stop()
	}
	n.role = Follower
	n.leaderID = ""
	n.persistent.CurrentTerm = term
	n.persistent.VotedFor = ""
	if err := n.persistLocked(); err != nil {
		log.Printf("persist metadata: %v", err)
	}

	if wasLeader {
		n.metrics.LeaderChanges++
	}

	if wasCandidate {
		n.metrics.ElectionsLost++
	}
}

// lastLogInfoLocked returns the index and term of the latest WAL entry.
// The caller must hold n.mu.
func (n *Node) lastLogInfoLocked() (
	lastIndex uint64,
	lastTerm uint64,
	err error,
) {
	entry, ok, err := n.wal.LastEntry()
	if err != nil {
		return 0, 0, err
	}

	if !ok {
		return 0, 0, nil
	}

	return uint64(entry.Index),
		uint64(entry.Term),
		nil
}

// HandleInstallSnapshot processes an incoming InstallSnapshot RPC.
func (n *Node) HandleInstallSnapshot(
	_ context.Context,
	req *pb.InstallSnapshotRequest,
) (*pb.InstallSnapshotResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// 1. Reply immediately if term < currentTerm
	if req.Term < n.persistent.CurrentTerm {
		return &pb.InstallSnapshotResponse{Term: n.persistent.CurrentTerm}, nil
	}

	n.electionTimer.reset()

	if req.Term > n.persistent.CurrentTerm {
		n.becomeFollowerLocked(req.Term)
	} else if n.role != Follower {
		n.role = Follower
	}

	// Save snapshot file, discard any existing or partial snapshot with a smaller index
	if req.LastIncludedIndex <= n.volatile.LastIncludedIndex {
		return &pb.InstallSnapshotResponse{Term: n.persistent.CurrentTerm}, nil
	}

	// Restore state machine
	if err := n.stateMachine.Restore(req.Data); err != nil {
		return nil, err
	}

	n.metrics.SnapshotsInstalled++

	// Update snapshot store
	snap := snapshot.Snapshot{
		LastIncludedIndex: req.LastIncludedIndex,
		LastIncludedTerm:  req.LastIncludedTerm,
		Data:              req.Data,
	}
	if err := n.snapshotStore.Save(snap); err != nil {
		return nil, err
	}

	n.volatile.LastIncludedIndex = req.LastIncludedIndex
	n.volatile.LastIncludedTerm = req.LastIncludedTerm

	// If existing log entry has same index and term as snapshot's last included entry, retain log entries following it and reply
	entry, ok, err := n.wal.EntryAt(req.LastIncludedIndex)
	if err == nil && ok && uint64(entry.Term) == req.LastIncludedTerm {
		_ = n.wal.TruncateBefore(req.LastIncludedIndex)
	} else {
		// Discard the entire log
		// We can do this by just truncating after 0, but since the API doesn't fully support wiping and resetting index easily,
		// we'll truncate after 0 (which removes everything). Wait, TruncateAfter(0) will keep entries <= 0, which is empty.
		_ = n.wal.TruncateAfter(0)
	}

	if n.volatile.CommitIndex < req.LastIncludedIndex {
		n.volatile.CommitIndex = req.LastIncludedIndex
	}
	if n.volatile.LastApplied < req.LastIncludedIndex {
		n.volatile.LastApplied = req.LastIncludedIndex
	}

	return &pb.InstallSnapshotResponse{Term: n.persistent.CurrentTerm}, nil
}
