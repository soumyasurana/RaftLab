package raft

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/soumyasurana/RaftLab/internal/config"
	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/internal/snapshot"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/internal/storage/metadata"
	"github.com/soumyasurana/RaftLab/internal/storage/wal"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

func createSnapshotTestConfig(dir string, threshold uint64) *config.Config {
	cfg := createTestConfig(dir)
	cfg.Node.SnapshotThreshold = threshold
	return cfg
}

func TestSnapshotCreationAndTruncation(t *testing.T) {
	dir := t.TempDir()
	cfg := createSnapshotTestConfig(dir, 2)

	node, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	node.mu.Lock()
	node.role = Leader
	node.mu.Unlock()

	// Propose 3 entries to trigger snapshot
	cmd1, _ := statemachine.Encode(statemachine.Command{Operation: statemachine.OpSet, Key: "k1", Value: "v1"})
	cmd2, _ := statemachine.Encode(statemachine.Command{Operation: statemachine.OpSet, Key: "k2", Value: "v2"})
	cmd3, _ := statemachine.Encode(statemachine.Command{Operation: statemachine.OpSet, Key: "k3", Value: "v3"})

	err = node.wal.AppendEntries([]types.LogEntry{
		{Index: 1, Term: 1, Command: cmd1},
		{Index: 2, Term: 1, Command: cmd2},
		{Index: 3, Term: 1, Command: cmd3},
	})
	if err != nil {
		t.Fatal(err)
	}

	node.mu.Lock()
	node.volatile.CommitIndex = 3
	node.mu.Unlock()

	err = node.applyCommittedEntries() // This will apply up to 3 and trigger maybeTakeSnapshot
	if err != nil {
		t.Fatal(err)
	}

	// Verify snapshot was taken
	snap, exists, err := node.snapshotStore.Load()
	if err != nil || !exists {
		t.Fatalf("Expected snapshot to exist")
	}

	if snap.LastIncludedIndex != 3 {
		t.Fatalf("Expected LastIncludedIndex 3, got %d", snap.LastIncludedIndex)
	}

	// Verify WAL was truncated
	entries, _ := node.wal.ReadAll()
	if len(entries) != 0 {
		t.Fatalf("Expected WAL to be empty after snapshot, got %d entries", len(entries))
	}
}

func TestHandleInstallSnapshot(t *testing.T) {
	dir := t.TempDir()
	cfg := createSnapshotTestConfig(dir, 100)

	node, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer node.Stop()

	// Create dummy snapshot state
	kv := statemachine.New()
	kv.Apply(statemachine.Command{Operation: statemachine.OpSet, Key: "k1", Value: "v1"})
	data, _ := kv.Marshal()

	req := &pb.InstallSnapshotRequest{
		Term:              2,
		LeaderId:          "node2",
		LastIncludedIndex: 5,
		LastIncludedTerm:  1,
		Data:              data,
	}

	resp, err := node.HandleInstallSnapshot(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Term != 2 {
		t.Fatalf("Expected term 2, got %d", resp.Term)
	}

	// Verify state
	if node.volatile.LastIncludedIndex != 5 {
		t.Fatalf("Expected LastIncludedIndex 5")
	}
	if node.volatile.CommitIndex != 5 {
		t.Fatalf("Expected CommitIndex 5")
	}
	if node.volatile.LastApplied != 5 {
		t.Fatalf("Expected LastApplied 5")
	}

	val, ok := node.stateMachine.Get("k1")
	if !ok || val != "v1" {
		t.Fatalf("Expected state machine to be updated")
	}
}

func TestRecoveryWithSnapshotAndWAL(t *testing.T) {
	dir := t.TempDir()
	cfg := createSnapshotTestConfig(dir, 10)

	// Pre-create a snapshot (Index=5) and a WAL with entries 6 and 7
	kv := statemachine.New()
	kv.Apply(statemachine.Command{Operation: statemachine.OpSet, Key: "k1", Value: "v1"})
	data, _ := kv.Marshal()

	snapStore := snapshot.NewFileStore(dir)
	snapStore.Save(snapshot.Snapshot{LastIncludedIndex: 5, LastIncludedTerm: 1, Data: data})

	m, _ := metadata.Open(filepath.Join(dir, "metadata.json"))
	m.Save(metadata.PersistentState{CurrentTerm: 2})

	w, _ := wal.Open(filepath.Join(dir, "raft.wal"))
	cmd2, _ := statemachine.Encode(statemachine.Command{Operation: statemachine.OpSet, Key: "k2", Value: "v2"})
	w.AppendEntries([]types.LogEntry{
		{Index: 6, Term: 2, Command: cmd2},
		{Index: 7, Term: 2, Command: cmd2},
	})
	w.Close()

	// Recover node
	node, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer node.Stop()

	// Volatile state is reset on restart.
	if node.volatile.CommitIndex != 5 {
		t.Fatalf("expected CommitIndex 5 after snapshot recovery, got %d", node.volatile.CommitIndex)
	}

	if node.volatile.LastApplied != 5 {
		t.Fatalf("expected LastApplied 5 after snapshot recovery, got %d", node.volatile.LastApplied)
	}

	// Snapshot state should already be restored.
	val, ok := node.stateMachine.Get("k1")
	if !ok || val != "v1" {
		t.Fatalf("expected k1=v1 from snapshot")
	}
}
