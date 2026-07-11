package raft

import (
	"context"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
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
		n.persistent.VotedFor = types.NodeID(req.CandidateId)

		// Later, will reset the election timer here.
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

	// Reject stale leaders.
	if req.Term < n.persistent.CurrentTerm {

		response := &pb.AppendEntriesResponse{
			Term:    n.persistent.CurrentTerm,
			Success: false,
		}

		n.mu.Unlock()

		return response, nil
	}

	// Update term / become follower.
	if req.Term > n.persistent.CurrentTerm {
		n.becomeFollowerLocked(req.Term)
	} else if n.role != Follower {
		n.role = Follower
	}

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

	response := &pb.AppendEntriesResponse{
		Term:    n.persistent.CurrentTerm,
		Success: true,
	}

	n.mu.Unlock()

	n.electionTimer.reset()

	return response, nil
}

// becomeFollowerLocked transitions the node to follower state.
// The caller must hold n.mu.
func (n *Node) becomeFollowerLocked(term uint64) {
	if n.heartbeat != nil {
		n.heartbeat.stop()
	}
	n.role = Follower
	n.persistent.CurrentTerm = term
	n.persistent.VotedFor = ""
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
