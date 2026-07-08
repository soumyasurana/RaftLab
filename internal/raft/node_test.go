package raft

import (
	"path/filepath"
	"testing"

	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

func TestNewNode(t *testing.T) {
	dir := t.TempDir()

	cfg := &config.Config{
		Node: types.NodeConfig{
			ID:      "node1",
			Address: "localhost:50051",
			DataDir: filepath.Join(dir),
		},
	}

	node, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if node.role != Follower {
		t.Fatal("new node should start as follower")
	}

	if node.wal == nil {
		t.Fatal("wal not initialized")
	}

	if node.stateMachine == nil {
		t.Fatal("state machine not initialized")
	}
}
