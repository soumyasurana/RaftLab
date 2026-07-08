package raft

import "github.com/soumyasurana/RaftLab/pkg/types"

type Role uint8

const (
	Follower Role = iota
	Candidate
	Leader
)

func (r Role) String() string {
	switch r {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

// PersistentState survives crashes.
type PersistentState struct {
	CurrentTerm uint64
	VotedFor    types.NodeID
}

// VolatileState exists only in memory.
type VolatileState struct {
	CommitIndex uint64
	LastApplied uint64
}
