package raft

import (
	"testing"
	"time"

	"github.com/soumyasurana/RaftLab/internal/config"
	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

func TestInitializeLeaderState(t *testing.T) {
	cfg := &config.Config{
		Node: types.NodeConfig{
			ID:               "node1",
			Address:          "localhost:50051",
			DataDir:          t.TempDir(),
			ElectionTimeout:  300 * time.Millisecond,
			HeartbeatTimeout: 100 * time.Millisecond,
			Peers: []types.Peer{
				{
					ID:      "node2",
					Address: "localhost:50052",
				},
				{
					ID:      "node3",
					Address: "localhost:50053",
				},
			},
		},
	}

	node, err := New(cfg)
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	defer func() {
		if err := node.Stop(); err != nil {
			t.Fatalf("stop node: %v", err)
		}
	}()

	const lastLogIndex uint64 = 17

	node.initializeLeaderState(lastLogIndex)

	if len(node.volatile.NextIndex) != len(cfg.Node.Peers) {
		t.Fatalf(
			"expected %d nextIndex entries, got %d",
			len(cfg.Node.Peers),
			len(node.volatile.NextIndex),
		)
	}

	if len(node.volatile.MatchIndex) != len(cfg.Node.Peers) {
		t.Fatalf(
			"expected %d matchIndex entries, got %d",
			len(cfg.Node.Peers),
			len(node.volatile.MatchIndex),
		)
	}

	for _, peer := range cfg.Node.Peers {
		next := node.volatile.NextIndex[peer.ID]
		match := node.volatile.MatchIndex[peer.ID]

		if next != lastLogIndex+1 {
			t.Fatalf(
				"peer %s: expected nextIndex=%d, got %d",
				peer.ID,
				lastLogIndex+1,
				next,
			)
		}

		if match != 0 {
			t.Fatalf(
				"peer %s: expected matchIndex=0, got %d",
				peer.ID,
				match,
			)
		}
	}
}

func TestHandleAppendEntriesResponseMonotonicUpdates(t *testing.T) {
	cfg := &config.Config{
		Node: types.NodeConfig{
			ID:      "node1",
			Address: "localhost:50051",
			DataDir: t.TempDir(),
			Peers: []types.Peer{
				{
					ID:      "node2",
					Address: "localhost:50052",
				},
			},
		},
	}

	node, err := New(cfg)
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	defer node.Stop()

	node.mu.Lock()
	node.role = Leader
	node.persistent.CurrentTerm = 1
	node.initializeLeaderState(0)
	node.mu.Unlock()

	// Suppose response for entries 1..5 arrives: prevIndex = 0, replicated = 5
	resp := &pb.AppendEntriesResponse{
		Term:    1,
		Success: true,
	}
	node.handleAppendEntriesResponse("node2", resp, 0, 5)

	node.mu.RLock()
	next := node.volatile.NextIndex["node2"]
	match := node.volatile.MatchIndex["node2"]
	node.mu.RUnlock()

	if match != 5 || next != 6 {
		t.Fatalf("expected match=5, next=6, got match=%d, next=%d", match, next)
	}

	// Suppose an earlier response (entries 1..3: prevIndex = 0, replicated = 3) arrives late
	node.handleAppendEntriesResponse("node2", resp, 0, 3)

	node.mu.RLock()
	next = node.volatile.NextIndex["node2"]
	match = node.volatile.MatchIndex["node2"]
	node.mu.RUnlock()

	// Should not regress or falsely add
	if match != 5 || next != 6 {
		t.Fatalf("expected match=5, next=6 to remain monotonic, got match=%d, next=%d", match, next)
	}
}

