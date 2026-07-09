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
	defer n.mu.Unlock()

	// Reject the leaders from an older term.
	if req.Term < n.persistent.CurrentTerm {
		return &pb.AppendEntriesResponse{
			Term:    n.persistent.CurrentTerm,
			Success: false,
		}, nil
	}

	// a valid AppendEntries request here means this node recognizes the sender as the leader for the current term.
	if req.Term > n.persistent.CurrentTerm {
		n.becomeFollowerLocked(req.Term)
	} else if n.role != Follower {
		n.role = Follower
	}

	/*
		Later:
		1. Reset the election timer.
		2. Verify PrevLogIndex and PrevLogTerm.
		3. Remove conflicting entries.
		4. Append new entries to the WAL.
		5. Advance CommitIndex.
		6. Apply committed entries to the state machine.
	*/
	return &pb.AppendEntriesResponse{
		Term:    n.persistent.CurrentTerm,
		Success: true,
	}, nil
}

// becomeFollowerLocked transitions the node to follower state.
// The caller must hold n.mu.
func (n *Node) becomeFollowerLocked(term uint64) {
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
	entries, err := n.wal.ReadAll()
	if err != nil {
		return 0, 0, err
	}

	if len(entries) == 0 {
		return 0, 0, nil
	}

	lastEntry := entries[len(entries)-1]

	return uint64(lastEntry.Index), uint64(lastEntry.Term), nil
}
