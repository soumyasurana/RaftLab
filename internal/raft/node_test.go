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

func TestNodeRestartRecoversMetadata(t *testing.T) {
	dir := t.TempDir()

	cfg := &config.Config{
		Node: types.NodeConfig{
			ID:      "node1",
			Address: "localhost:50051",
			DataDir: dir,
		},
	}

	node1, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create node1: %v", err)
	}

	// Mutate state and persist
	node1.mu.Lock()
	node1.persistent.CurrentTerm = 42
	node1.persistent.VotedFor = "node2"
	if err := node1.persistLocked(); err != nil {
		t.Fatalf("Failed to persist node1 state: %v", err)
	}
	node1.mu.Unlock()

	// Stop node1 to simulate crash/restart
	_ = node1.Stop()

	// Start node2 with the same directory
	node2, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create node2: %v", err)
	}

	node2.mu.RLock()
	term := node2.persistent.CurrentTerm
	votedFor := node2.persistent.VotedFor
	node2.mu.RUnlock()

	if term != 42 {
		t.Fatalf("Expected CurrentTerm 42, got %d", term)
	}

	if votedFor != "node2" {
		t.Fatalf("Expected VotedFor node2, got %s", votedFor)
	}

	_ = node2.Stop()
}

func TestNodeStopIdempotent(t *testing.T) {
	dir := t.TempDir()

	cfg := &config.Config{
		Node: types.NodeConfig{
			ID:      "node1",
			Address: "localhost:50051",
			DataDir: dir,
		},
	}

	node, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	// First stop
	if err := node.Stop(); err != nil {
		t.Fatalf("first Stop returned error: %v", err)
	}

	// Second stop should not panic or fail
	if err := node.Stop(); err != nil {
		t.Fatalf("second Stop returned error: %v", err)
	}
}

