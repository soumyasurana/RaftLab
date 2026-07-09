package raft

import (
	"context"
	"testing"

	"github.com/soumyasurana/RaftLab/internal/config"
	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

func newTestNode(t *testing.T) *Node {
	t.Helper()

	cfg := &config.Config{}

	cfg.Node.ID = types.NodeID("node-1")
	cfg.Node.Address = "127.0.0.1:50051"
	cfg.Node.DataDir = t.TempDir()

	node, err := New(cfg)
	if err != nil {
		t.Fatalf("create test node: %v", err)
	}

	t.Cleanup(func() {
		if err := node.Stop(); err != nil {
			t.Errorf("stop test node: %v", err)
		}
	})

	return node
}

func TestRequestVoteRejectsOlderTerm(t *testing.T) {
	node := newTestNode(t)

	node.persistent.CurrentTerm = 5

	response, err := node.HandleRequestVote(
		context.Background(),
		&pb.RequestVoteRequest{
			Term:         4,
			CandidateId:  "node-2",
			LastLogIndex: 0,
			LastLogTerm:  0,
		},
	)
	if err != nil {
		t.Fatalf("handle RequestVote: %v", err)
	}

	if response.VoteGranted {
		t.Fatal("expected vote to be rejected")
	}

	if response.Term != 5 {
		t.Fatalf(
			"expected response term 5, got %d",
			response.Term,
		)
	}
}

func TestRequestVoteGrantsVoteToEligibleCandidate(t *testing.T) {
	node := newTestNode(t)

	response, err := node.HandleRequestVote(
		context.Background(),
		&pb.RequestVoteRequest{
			Term:         1,
			CandidateId:  "node-2",
			LastLogIndex: 0,
			LastLogTerm:  0,
		},
	)
	if err != nil {
		t.Fatalf("handle RequestVote: %v", err)
	}

	if !response.VoteGranted {
		t.Fatal("expected vote to be granted")
	}

	if response.Term != 1 {
		t.Fatalf(
			"expected response term 1, got %d",
			response.Term,
		)
	}

	if node.persistent.CurrentTerm != 1 {
		t.Fatalf(
			"expected current term 1, got %d",
			node.persistent.CurrentTerm,
		)
	}

	if node.persistent.VotedFor != types.NodeID("node-2") {
		t.Fatalf(
			"expected vote for node-2, got %s",
			node.persistent.VotedFor,
		)
	}
}

func TestRequestVoteRejectsSecondCandidateInSameTerm(
	t *testing.T,
) {
	node := newTestNode(t)

	node.persistent.CurrentTerm = 3
	node.persistent.VotedFor = types.NodeID("node-2")

	response, err := node.HandleRequestVote(
		context.Background(),
		&pb.RequestVoteRequest{
			Term:         3,
			CandidateId:  "node-3",
			LastLogIndex: 0,
			LastLogTerm:  0,
		},
	)
	if err != nil {
		t.Fatalf("handle RequestVote: %v", err)
	}

	if response.VoteGranted {
		t.Fatal(
			"expected vote for second candidate to be rejected",
		)
	}

	if node.persistent.VotedFor != types.NodeID("node-2") {
		t.Fatalf(
			"expected existing vote for node-2 to remain, got %s",
			node.persistent.VotedFor,
		)
	}
}

func TestRequestVoteAllowsSameCandidateAgain(t *testing.T) {
	node := newTestNode(t)

	node.persistent.CurrentTerm = 3
	node.persistent.VotedFor = types.NodeID("node-2")

	response, err := node.HandleRequestVote(
		context.Background(),
		&pb.RequestVoteRequest{
			Term:         3,
			CandidateId:  "node-2",
			LastLogIndex: 0,
			LastLogTerm:  0,
		},
	)
	if err != nil {
		t.Fatalf("handle RequestVote: %v", err)
	}

	if !response.VoteGranted {
		t.Fatal(
			"expected repeated request from same candidate to succeed",
		)
	}
}

func TestRequestVoteRejectsStaleCandidateLog(t *testing.T) {
	node := newTestNode(t)

	err := node.wal.Append(types.LogEntry{
		Index:   1,
		Term:    4,
		Command: []byte("SET name Soumya"),
	})
	if err != nil {
		t.Fatalf("append WAL entry: %v", err)
	}

	response, err := node.HandleRequestVote(
		context.Background(),
		&pb.RequestVoteRequest{
			Term:         5,
			CandidateId:  "node-2",
			LastLogIndex: 10,
			LastLogTerm:  3,
		},
	)
	if err != nil {
		t.Fatalf("handle RequestVote: %v", err)
	}

	if response.VoteGranted {
		t.Fatal(
			"expected candidate with an older log term to be rejected",
		)
	}
}

func TestAppendEntriesRejectsOlderTerm(t *testing.T) {
	node := newTestNode(t)

	node.persistent.CurrentTerm = 5

	response, err := node.HandleAppendEntries(
		context.Background(),
		&pb.AppendEntriesRequest{
			Term:     4,
			LeaderId: "node-2",
		},
	)
	if err != nil {
		t.Fatalf("handle AppendEntries: %v", err)
	}

	if response.Success {
		t.Fatal(
			"expected AppendEntries from an older term to fail",
		)
	}

	if response.Term != 5 {
		t.Fatalf(
			"expected response term 5, got %d",
			response.Term,
		)
	}
}

func TestAppendEntriesUpdatesTermAndBecomesFollower(
	t *testing.T,
) {
	node := newTestNode(t)

	node.role = Candidate
	node.persistent.CurrentTerm = 2
	node.persistent.VotedFor = types.NodeID("node-1")

	response, err := node.HandleAppendEntries(
		context.Background(),
		&pb.AppendEntriesRequest{
			Term:     3,
			LeaderId: "node-2",
		},
	)
	if err != nil {
		t.Fatalf("handle AppendEntries: %v", err)
	}

	if !response.Success {
		t.Fatal("expected AppendEntries to succeed")
	}

	if node.role != Follower {
		t.Fatalf(
			"expected follower role, got %s",
			node.role,
		)
	}

	if node.persistent.CurrentTerm != 3 {
		t.Fatalf(
			"expected current term 3, got %d",
			node.persistent.CurrentTerm,
		)
	}

	if node.persistent.VotedFor != "" {
		t.Fatalf(
			"expected vote to be cleared, got %s",
			node.persistent.VotedFor,
		)
	}
}
