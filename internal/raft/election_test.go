package raft

import (
	"testing"
	"time"

	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

func TestSingleNodeElectsItselfLeader(t *testing.T) {
	cfg := &config.Config{
		Node: types.NodeConfig{
			ID:               "node-1",
			Address:          "127.0.0.1:50051",
			DataDir:          t.TempDir(),
			ElectionTimeout:  50 * time.Millisecond,
			HeartbeatTimeout: 10 * time.Millisecond,
		},
	}

	node, err := New(cfg)
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	t.Cleanup(func() {
		if err := node.Stop(); err != nil {
			t.Errorf("stop node: %v", err)
		}
	})

	node.handleElectionTimeout()

	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.role != Leader {
		t.Fatalf(
			"expected role %s, got %s",
			Leader,
			node.role,
		)
	}

	if node.persistent.CurrentTerm != 1 {
		t.Fatalf(
			"expected term 1, got %d",
			node.persistent.CurrentTerm,
		)
	}

	if node.persistent.VotedFor != cfg.Node.ID {
		t.Fatalf(
			"expected self-vote for %s, got %s",
			cfg.Node.ID,
			node.persistent.VotedFor,
		)
	}
}

func TestElectionTimeoutStartsNewTerm(t *testing.T) {
	node := newElectionTestNode(t)

	node.handleElectionTimeout()

	node.mu.RLock()

	firstTerm := node.persistent.CurrentTerm
	firstRole := node.role

	node.mu.RUnlock()

	if firstTerm != 1 {
		t.Fatalf(
			"expected first election term 1, got %d",
			firstTerm,
		)
	}

	if firstRole != Candidate {
		t.Fatalf(
			"expected candidate role, got %s",
			firstRole,
		)
	}

	node.handleElectionTimeout()

	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.persistent.CurrentTerm != 2 {
		t.Fatalf(
			"expected second election term 2, got %d",
			node.persistent.CurrentTerm,
		)
	}

	if node.persistent.VotedFor != node.config.Node.ID {
		t.Fatalf(
			"expected node to vote for itself, got %s",
			node.persistent.VotedFor,
		)
	}
}

func TestLeaderIgnoresElectionTimeout(t *testing.T) {
	node := newElectionTestNode(t)

	node.mu.Lock()
	node.role = Leader
	node.persistent.CurrentTerm = 4
	node.mu.Unlock()

	node.handleElectionTimeout()

	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.role != Leader {
		t.Fatalf(
			"expected role to remain leader, got %s",
			node.role,
		)
	}

	if node.persistent.CurrentTerm != 4 {
		t.Fatalf(
			"expected term to remain 4, got %d",
			node.persistent.CurrentTerm,
		)
	}
}

func TestBecomeLeaderRejectsStaleElection(t *testing.T) {
	node := newElectionTestNode(t)

	node.mu.Lock()
	node.role = Candidate
	node.persistent.CurrentTerm = 5
	node.mu.Unlock()

	node.becomeLeader(4)

	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.role != Candidate {
		t.Fatalf(
			"expected stale election to be ignored, got role %s",
			node.role,
		)
	}
}

func TestBecomeLeaderPromotesCurrentCandidate(t *testing.T) {
	node := newElectionTestNode(t)

	node.mu.Lock()
	node.role = Candidate
	node.persistent.CurrentTerm = 5
	node.mu.Unlock()

	node.becomeLeader(5)

	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.role != Leader {
		t.Fatalf(
			"expected candidate to become leader, got %s",
			node.role,
		)
	}
}

func newElectionTestNode(t *testing.T) *Node {
	t.Helper()

	cfg := &config.Config{
		Node: types.NodeConfig{
			ID:               "node-1",
			Address:          "127.0.0.1:50051",
			DataDir:          t.TempDir(),
			ElectionTimeout:  100 * time.Millisecond,
			HeartbeatTimeout: 25 * time.Millisecond,

			// This peer is intentionally unreachable.
			// The node cannot obtain quorum and remains a candidate.
			Peers: []types.Peer{
				{
					ID:      "node-2",
					Address: "127.0.0.1:65534",
				},
			},
		},
	}

	node, err := New(cfg)
	if err != nil {
		t.Fatalf("create election test node: %v", err)
	}

	t.Cleanup(func() {
		if err := node.Stop(); err != nil {
			t.Errorf("stop election test node: %v", err)
		}
	})

	return node
}
