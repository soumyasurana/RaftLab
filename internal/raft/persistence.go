package raft

import (
	"github.com/soumyasurana/RaftLab/internal/storage/metadata"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

// persistLocked persists the Raft persistent state.
//
// The caller must hold n.mu.
func (n *Node) persistLocked() error {
	return n.metadata.Save(
		metadata.PersistentState{
			CurrentTerm: types.Term(n.persistent.CurrentTerm),
			VotedFor:    n.persistent.VotedFor,
		},
	)
}
