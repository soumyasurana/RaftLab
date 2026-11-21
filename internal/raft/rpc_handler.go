package raft

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/internal/snapshot"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

// HandleRequestVote processes an incoming RequestVote RPC.
func (n *Node) HandleRequestVote(
	_ context.Context,
	req *pb.RequestVoteRequest,
) (*pb.RequestVoteResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.stopped {
		return nil, status.Error(codes.Unavailable, "node is stopped")
	}

	// Reject candidates from an older term.
	if req.Term < n.persistent.CurrentTerm {
		return &pb.RequestVoteResponse{
			Term:        n.persistent.CurrentTerm,
			VoteGranted: false,
		}, nil
	}

	// A newer term always causes this node to become a follower.
	if req.Term > n.persistent.CurrentTerm {
		if err := n.becomeFollowerLocked(req.Term); err != nil {
			return nil, err
		}
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
		previousVote := n.persistent.VotedFor
		n.persistent.VotedFor = types.NodeID(req.CandidateId)
		if err := n.persistLocked(); err != nil {
			n.persistent.VotedFor = previousVote
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
	defer n.mu.Unlock()
	if n.stopped {
		return nil, status.Error(codes.Unavailable, "node is stopped")
	}
	n.metrics.AppendEntriesRecv++

	rejected := &pb.AppendEntriesResponse{Term: n.persistent.CurrentTerm}
	if req.Term < n.persistent.CurrentTerm {
		return rejected, nil
	}

	if err := n.becomeFollowerLocked(req.Term); err != nil {
		return nil, err
	}
	n.leaderID = types.NodeID(req.LeaderId)
	n.electionTimer.reset()
	rejected.Term = n.persistent.CurrentTerm

	if req.PrevLogIndex < n.volatile.LastIncludedIndex {
		return rejected, nil
	}
	if req.PrevLogIndex == n.volatile.LastIncludedIndex {
		if req.PrevLogTerm != n.volatile.LastIncludedTerm {
			return rejected, nil
		}
	} else {
		entry, ok, err := n.wal.EntryAt(req.PrevLogIndex)
		if err != nil {
			return nil, err
		}
		if !ok || uint64(entry.Term) != req.PrevLogTerm {
			return rejected, nil
		}
	}

	matchedIndex := req.PrevLogIndex
	for _, entry := range req.Entries {
		if entry == nil || entry.Index != matchedIndex+1 {
			return nil, status.Error(codes.InvalidArgument, "log entries must be contiguous")
		}
		matchedIndex = entry.Index
	}
	if err := n.resolveConflicts(protobufEntriesToLogEntries(req.Entries)); err != nil {
		return nil, err
	}

	// Only commit the prefix confirmed by this RPC. The follower may still
	// have divergent entries beyond it.
	if commitIndex := min(req.LeaderCommit, matchedIndex); commitIndex > n.volatile.CommitIndex {
		n.volatile.CommitIndex = commitIndex
	}
	if n.volatile.LastApplied < n.volatile.CommitIndex {
		if err := n.applyCommittedEntries(); err != nil {
			return nil, err
		}
	}

	return &pb.AppendEntriesResponse{Term: n.persistent.CurrentTerm, Success: true}, nil
}

// becomeFollowerLocked transitions the node to follower state.
// The caller must hold n.mu.
func (n *Node) becomeFollowerLocked(term uint64) error {
	if term > n.persistent.CurrentTerm {
		previous := n.persistent
		n.persistent = PersistentState{CurrentTerm: term}
		if err := n.persistLocked(); err != nil {
			n.persistent = previous
			return err
		}
		n.leaderID = ""
	}

	switch n.role {
	case Leader:
		n.heartbeat.stop()
		n.metrics.LeaderChanges++
	case Candidate:
		n.metrics.ElectionsLost++
	}
	n.role = Follower
	return nil
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
		return n.volatile.LastIncludedIndex, n.volatile.LastIncludedTerm, nil
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
	if n.stopped {
		return nil, status.Error(codes.Unavailable, "node is stopped")
	}

	if req.Term < n.persistent.CurrentTerm {
		return &pb.InstallSnapshotResponse{Term: n.persistent.CurrentTerm}, nil
	}
	if err := n.becomeFollowerLocked(req.Term); err != nil {
		return nil, err
	}
	n.leaderID = types.NodeID(req.LeaderId)
	n.electionTimer.reset()
	response := &pb.InstallSnapshotResponse{Term: n.persistent.CurrentTerm}

	if req.LastIncludedIndex <= n.volatile.LastApplied {
		return response, nil
	}

	restored := statemachine.New()
	if err := restored.Restore(req.Data); err != nil {
		return nil, err
	}
	entry, ok, err := n.wal.EntryAt(req.LastIncludedIndex)
	if err != nil {
		return nil, err
	}
	if !ok || uint64(entry.Term) != req.LastIncludedTerm {
		if req.LastIncludedIndex <= n.volatile.CommitIndex {
			return nil, fmt.Errorf("snapshot conflicts with committed log at index %d", req.LastIncludedIndex)
		}
		// Keep the committed prefix recoverable if saving the snapshot fails.
		if err := n.wal.TruncateAfter(n.volatile.CommitIndex); err != nil {
			return nil, err
		}
	}

	snap := snapshot.Snapshot{
		LastIncludedIndex: req.LastIncludedIndex,
		LastIncludedTerm:  req.LastIncludedTerm,
		Data:              req.Data,
	}
	if err := n.snapshotStore.Save(snap); err != nil {
		return nil, err
	}

	n.stateMachine = restored
	n.volatile.LastIncludedIndex = req.LastIncludedIndex
	n.volatile.LastIncludedTerm = req.LastIncludedTerm
	n.volatile.LastApplied = req.LastIncludedIndex
	n.volatile.CommitIndex = max(n.volatile.CommitIndex, req.LastIncludedIndex)
	n.metrics.SnapshotsInstalled++

	if err := n.wal.TruncateBefore(req.LastIncludedIndex); err != nil {
		return nil, err
	}
	return response, n.applyCommittedEntries()
}
