package raft

import "github.com/soumyasurana/RaftLab/pkg/types"

// resolveConflicts reconciles the follower log with the leader's log.
// The caller must hold n.mu.
func (n *Node) resolveConflicts(
	incoming []types.LogEntry,
) error {

	for i, incomingEntry := range incoming {

		localEntry, ok, err := n.wal.EntryAt(
			uint64(incomingEntry.Index),
		)
		if err != nil {
			return err
		}

		// No local entry at this index.
		// Append all remaining leader entries.
		if !ok {
			return n.wal.AppendEntries(incoming[i:])
		}

		// Terms differ -> follower log diverged.
		if localEntry.Term != incomingEntry.Term {

			if err := n.wal.TruncateAfter(
				uint64(incomingEntry.Index - 1),
			); err != nil {
				return err
			}

			return n.wal.AppendEntries(incoming[i:])
		}

		// Same term.
		// Continue checking.
	}

	return nil
}
