package raft

import (
	"fmt"

	"github.com/soumyasurana/RaftLab/internal/statemachine"
)

// applyCommittedEntries applies every committed entry exactly once.
//
// The caller must hold n.mu.
func (n *Node) applyCommittedEntries() error {

	if n.volatile.LastApplied < n.volatile.LastIncludedIndex {
		n.volatile.LastApplied = n.volatile.LastIncludedIndex
	}

	for n.volatile.LastApplied < n.volatile.CommitIndex {

		nextIndex := n.volatile.LastApplied + 1

		entry, ok, err := n.wal.EntryAt(nextIndex)
		if err != nil {
			return err
		}

		if !ok {
			return fmt.Errorf("missing committed log entry at index %d", nextIndex)
		}

		command, err := statemachine.Decode(entry.Command)
		if err != nil {
			return err
		}

		if err := n.stateMachine.Apply(command); err != nil {
			return err
		}

		n.volatile.LastApplied = nextIndex
	}

	n.maybeTakeSnapshot()

	return nil
}
