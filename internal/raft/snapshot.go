package raft

import (
	"log"

	"github.com/soumyasurana/RaftLab/internal/snapshot"
)

// maybeTakeSnapshot checks if the log has grown beyond the threshold.
// If it has, it takes a snapshot and compacts the log.
// The caller must hold n.mu.
func (n *Node) maybeTakeSnapshot() {
	if n.config.Node.SnapshotThreshold == 0 {
		return // Snapshotting disabled
	}

	// We only compact the committed part of the log
	if n.volatile.CommitIndex <= n.volatile.LastIncludedIndex {
		return
	}

	// Count number of committed log entries since last snapshot
	logSize := n.volatile.CommitIndex - n.volatile.LastIncludedIndex

	if logSize >= n.config.Node.SnapshotThreshold {
		if err := n.takeSnapshot(); err != nil {
			log.Printf("failed to take snapshot: %v", err)
		}
	}
}

// takeSnapshot performs the actual snapshotting process.
// The caller must hold n.mu.
func (n *Node) takeSnapshot() error {
	log.Printf("Taking snapshot at index %d", n.volatile.LastApplied)

	// We snapshot the state up to LastApplied.
	// Since LastApplied might lag behind CommitIndex, we use LastApplied.
	snapshotIndex := n.volatile.LastApplied
	if snapshotIndex <= n.volatile.LastIncludedIndex {
		return nil
	}

	entry, ok, err := n.wal.EntryAt(snapshotIndex)
	if err != nil {
		return err
	}
	if !ok {
		return nil // Should not happen for a committed/applied entry
	}

	snapshotTerm := uint64(entry.Term)

	// Serialize the state machine
	data, err := n.stateMachine.Marshal()
	if err != nil {
		return err
	}

	snap := snapshot.Snapshot{
		LastIncludedIndex: snapshotIndex,
		LastIncludedTerm:  snapshotTerm,
		Data:              data,
	}

	if err := n.snapshotStore.Save(snap); err != nil {
		return err
	}

	n.volatile.LastIncludedIndex = snapshotIndex
	n.volatile.LastIncludedTerm = snapshotTerm

	if err := n.wal.TruncateBefore(snapshotIndex); err != nil {
		return err
	}

	log.Printf("Snapshot taken and WAL truncated up to index %d", snapshotIndex)
	return nil
}
