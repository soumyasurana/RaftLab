package raft

import (
	"sync"

	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/internal/storage/wal"
)

type Node struct {
	mu sync.RWMutex

	config *config.Config

	role Role

	persistent PersistentState
	volatile   VolatileState

	wal *wal.WAL

	stateMachine *statemachine.KVStore

	stopCh chan struct{}
}
