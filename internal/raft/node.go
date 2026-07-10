package raft

import (
	"sync"

	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/internal/rpc"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/internal/storage/wal"
)

type Node struct {
	mu            sync.RWMutex
	rpcClient     *rpc.Client
	config        *config.Config
	electionTimer *electionTimer
	role          Role
	heartbeat     *heartbeatManager
	persistent    PersistentState
	volatile      VolatileState

	wal *wal.WAL

	stateMachine *statemachine.KVStore

	stopCh chan struct{}
}

func (n *Node) VolatileState() {
	panic("unimplemented")
}
