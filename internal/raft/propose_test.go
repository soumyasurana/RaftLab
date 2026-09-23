package raft

import (
	"testing"

	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

func TestProposeAppendsEntry(t *testing.T) {
	cfg := &config.Config{
		Node: types.NodeConfig{
			ID:      "node1",
			Address: "localhost:50051",
			DataDir: t.TempDir(),
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

	node.role = Leader
	node.persistent.CurrentTerm = 1
	node.initializeLeaderState(0)

	cmd := statemachine.Command{
		Operation: statemachine.OpSet,
		Key:       "foo",
		Value:     "bar",
	}

	if err := node.Propose(cmd); err != nil {
		t.Fatalf("propose command: %v", err)
	}

	entries, err := node.wal.ReadAll()
	if err != nil {
		t.Fatalf("read wal: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 WAL entry, got %d", len(entries))
	}

	if entries[0].Index != 1 {
		t.Fatalf("expected index 1, got %d", entries[0].Index)
	}

	if entries[0].Term != 1 {
		t.Fatalf("expected term 1, got %d", entries[0].Term)
	}

	command, err := statemachine.DecodeCommand(entries[0].Command)
	if err != nil {
		t.Fatalf("decode command: %v", err)
	}

	if command.Operation != statemachine.OpSet {
		t.Fatal("unexpected operation")
	}

	if command.Key != "foo" {
		t.Fatalf("expected key foo, got %q", command.Key)
	}

	if command.Value != "bar" {
		t.Fatalf("expected value bar, got %q", command.Value)
	}
}

func TestProposeRejectsFollower(t *testing.T) {
	cfg := &config.Config{
		Node: types.NodeConfig{
			ID:      "node1",
			Address: "localhost:50051",
			DataDir: t.TempDir(),
		},
	}

	node, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	defer node.Stop()

	node.role = Follower

	cmd := statemachine.Command{
		Operation: statemachine.OpSet,
		Key:       "foo",
		Value:     "bar",
	}

	if err := node.Propose(cmd); err != ErrNotLeader {
		t.Fatalf("expected ErrNotLeader, got %v", err)
	}
}

func TestSingleNodeProposeCommitsAndApplies(t *testing.T) {
	cfg := &config.Config{
		Node: types.NodeConfig{
			ID:      "node1",
			Address: "localhost:50051",
			DataDir: t.TempDir(),
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

	cmd := statemachine.Command{
		Operation: statemachine.OpSet,
		Key:       "greeting",
		Value:     "hello world",
	}

	if err := node.Propose(cmd); err != nil {
		t.Fatalf("propose command: %v", err)
	}

	node.mu.RLock()
	commitIndex := node.volatile.CommitIndex
	lastApplied := node.volatile.LastApplied
	node.mu.RUnlock()

	if commitIndex != 1 {
		t.Fatalf("expected CommitIndex 1, got %d", commitIndex)
	}

	if lastApplied != 1 {
		t.Fatalf("expected LastApplied 1, got %d", lastApplied)
	}

	val, ok := node.stateMachine.Get("greeting")
	if !ok || val != "hello world" {
		t.Fatalf("expected state machine to have greeting='hello world', got val=%q, ok=%v", val, ok)
	}
}

