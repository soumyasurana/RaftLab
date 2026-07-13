package raft

// advanceCommitIndex advances the leader's CommitIndex.
//
// The caller must hold n.mu.
func (n *Node) advanceCommitIndex() {

	lastEntry, ok, err := n.wal.LastEntry()
	if err != nil || !ok {
		return
	}

	lastIndex := uint64(lastEntry.Index)

	clusterSize := len(n.config.Node.Peers) + 1
	quorum := clusterSize/2 + 1

	for index := lastIndex; index > n.volatile.CommitIndex; index-- {

		// Count the leader.
		replicated := 1

		for _, peer := range n.config.Node.Peers {

			if n.volatile.MatchIndex[peer.ID] >= index {
				replicated++
			}
		}

		if replicated < quorum {
			continue
		}

		entry, ok, err := n.wal.EntryAt(index)
		if err != nil || !ok {
			return
		}

		// Raft §5.4.2:
		// A leader only commits entries from its current term by counting replicas.
		if uint64(entry.Term) != n.persistent.CurrentTerm {
			continue
		}

		n.volatile.CommitIndex = index

		if err := n.applyCommittedEntries(); err != nil {
			return
		}

		return
	}
}
