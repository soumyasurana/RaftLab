package raft

import (
	"errors"

	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

var ErrNotLeader = errors.New("node is not the leader")

// Propose appends a client command to the Raft log.
//
// The leader appends the entry locally. Replication is initiated
// immediately afterwards.
func (n *Node) Propose(cmd statemachine.Command) error {

	n.mu.Lock()

	if n.stopped || n.role != Leader {
		n.mu.Unlock()
		return ErrNotLeader
	}

	if err := cmd.Validate(); err != nil {
		n.mu.Unlock()
		return err
	}

	payload, err := cmd.Encode()
	if err != nil {
		n.mu.Unlock()
		return err
	}

	lastEntry, ok, err := n.wal.LastEntry()
	if err != nil {
		n.mu.Unlock()
		return err
	}

	nextIndex := n.volatile.LastIncludedIndex + 1

	if ok {
		nextIndex = uint64(lastEntry.Index) + 1
	}

	entry := types.LogEntry{
		Index:   types.LogIndex(nextIndex),
		Term:    types.Term(n.persistent.CurrentTerm),
		Command: payload,
	}

	if err := n.wal.Append(entry); err != nil {
		n.mu.Unlock()
		return err
	}

	n.advanceCommitIndex()
	term := n.persistent.CurrentTerm
	n.mu.Unlock()

	n.replicateToPeers(term)

	return nil
}
