package raft

import (
	"testing"
	"time"

	"github.com/soumyasurana/RaftLab/internal/config"
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
