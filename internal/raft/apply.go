package raft

import "github.com/soumyasurana/RaftLab/internal/statemachine"

// applyCommittedEntries applies every committed entry exactly once.
//
// The caller must hold n.mu.
func (n *Node) applyCommittedEntries() error {

	for n.volatile.LastApplied < n.volatile.CommitIndex {

		nextIndex := n.volatile.LastApplied + 1

		entry, ok, err := n.wal.EntryAt(nextIndex)
		if err != nil {
			return err
		}

		if !ok {
			break
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
