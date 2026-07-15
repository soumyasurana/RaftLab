package raft

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/soumyasurana/RaftLab/internal/config"
	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/internal/storage/metadata"
	"github.com/soumyasurana/RaftLab/internal/storage/wal"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

func createTestConfig(dir string) *config.Config {
	return &config.Config{
		Node: types.NodeConfig{
			ID:               "node1",
			Address:          "localhost:50051",
			DataDir:          dir,
			HeartbeatTimeout: 50 * time.Millisecond,
			ElectionTimeout:  150 * time.Millisecond,
		},
	}
}

// restart with empty WAL
func TestRecoveryEmptyWAL(t *testing.T) {
	dir := t.TempDir()
	cfg := createTestConfig(dir)

	node1, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	if err := node1.Stop(); err != nil {
		t.Fatalf("Failed to stop node: %v", err)
	}

	node2, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to recover node: %v", err)
	}
	defer node2.Stop()

	if node2.volatile.CommitIndex != 0 {
		t.Fatalf("Expected CommitIndex 0, got %d", node2.volatile.CommitIndex)
	}
	if node2.volatile.LastApplied != 0 {
		t.Fatalf("Expected LastApplied 0, got %d", node2.volatile.LastApplied)
	}
}

// restart with populated WAL but no committed entries
func TestRecoveryPopulatedWAL(t *testing.T) {
	dir := t.TempDir()
	cfg := createTestConfig(dir)

	// Pre-populate WAL
	w, err := wal.Open(filepath.Join(dir, "raft.wal"))
	if err != nil {
		t.Fatal(err)
	}

	cmd, _ := statemachine.Encode(statemachine.Command{
		Operation: statemachine.OpSet,
		Key:       "key1",
		Value:     "value1",
	})

	err = w.AppendEntries([]types.LogEntry{
		{Index: 1, Term: 1, Command: cmd},
		{Index: 2, Term: 1, Command: cmd},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.Close()

	// Recover node
	node, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to recover node: %v", err)
	}
	defer node.Stop()

	// WAL should be loaded
	last, ok, _ := node.wal.LastEntry()
	if !ok || last.Index != 2 {
		t.Fatalf("Expected last index 2, got %d", last.Index)
	}

	// But CommitIndex and LastApplied are 0
	if node.volatile.CommitIndex != 0 {
		t.Fatalf("Expected CommitIndex 0, got %d", node.volatile.CommitIndex)
	}
	if node.volatile.LastApplied != 0 {
		t.Fatalf("Expected LastApplied 0, got %d", node.volatile.LastApplied)
	}

	// State machine should be empty
	if _, ok := node.stateMachine.Get("key1"); ok {
		t.Fatalf("Expected state machine to be empty")
	}
}

// restart with multiple committed entries, node catches up after restart
func TestRecoveryCatchesUpFromLeader(t *testing.T) {
	dir := t.TempDir()
	cfg := createTestConfig(dir)

	// Pre-populate WAL and Metadata
	w, err := wal.Open(filepath.Join(dir, "raft.wal"))
	if err != nil {
		t.Fatal(err)
	}

	cmd1, _ := statemachine.Encode(statemachine.Command{Operation: statemachine.OpSet, Key: "k1", Value: "v1"})
	cmd2, _ := statemachine.Encode(statemachine.Command{Operation: statemachine.OpSet, Key: "k2", Value: "v2"})

	_ = w.AppendEntries([]types.LogEntry{
		{Index: 1, Term: 1, Command: cmd1},
		{Index: 2, Term: 1, Command: cmd2},
	})
	w.Close()

	m, _ := metadata.Open(filepath.Join(dir, "metadata.json"))
	_ = m.Save(metadata.PersistentState{CurrentTerm: 1, VotedFor: "node2"})

	// Recover node
	node, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer node.Stop()

	// Simulate leader heartbeat with LeaderCommit = 2
	req := &pb.AppendEntriesRequest{
		Term:         1,
		LeaderId:     "node2",
		PrevLogIndex: 2,
		PrevLogTerm:  1,
		Entries:      nil,
		LeaderCommit: 2,
	}

	resp, err := node.HandleAppendEntries(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("Expected successful AppendEntries")
	}

	// Verify CommitIndex and LastApplied advanced
	if node.volatile.CommitIndex != 2 {
		t.Fatalf("Expected CommitIndex 2, got %d", node.volatile.CommitIndex)
	}
	if node.volatile.LastApplied != 2 {
		t.Fatalf("Expected LastApplied 2, got %d", node.volatile.LastApplied)
	}

	// Verify state machine has the values
	if val, ok := node.stateMachine.Get("k1"); !ok || val != "v1" {
		t.Fatalf("Expected k1=v1, got %v", val)
	}
	if val, ok := node.stateMachine.Get("k2"); !ok || val != "v2" {
		t.Fatalf("Expected k2=v2, got %v", val)
	}
}

// recovery idempotency (restart twice)
func TestRecoveryIdempotency(t *testing.T) {
	dir := t.TempDir()
	cfg := createTestConfig(dir)

	// Pre-populate WAL and Metadata
	w, err := wal.Open(filepath.Join(dir, "raft.wal"))
	if err != nil {
		t.Fatal(err)
	}

	cmd, _ := statemachine.Encode(statemachine.Command{Operation: statemachine.OpSet, Key: "k1", Value: "v1"})
	_ = w.AppendEntries([]types.LogEntry{{Index: 1, Term: 1, Command: cmd}})
	w.Close()

	m, _ := metadata.Open(filepath.Join(dir, "metadata.json"))
	_ = m.Save(metadata.PersistentState{CurrentTerm: 2, VotedFor: "node2"})

	// First recovery
	node1, _ := New(cfg)
	if node1.persistent.CurrentTerm != 2 {
		t.Fatalf("Expected term 2")
	}
	_ = node1.Stop()

	// Second recovery
	node2, _ := New(cfg)
	if node2.persistent.CurrentTerm != 2 {
		t.Fatalf("Expected term 2")
	}

	last, ok, _ := node2.wal.LastEntry()
	if !ok || last.Index != 1 {
		t.Fatalf("Expected last index 1")
	}
	_ = node2.Stop()
}
