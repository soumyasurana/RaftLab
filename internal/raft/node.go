package raft

import (
	"sync"

	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/internal/rpc"
	"github.com/soumyasurana/RaftLab/internal/snapshot"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/internal/storage/metadata"
	"github.com/soumyasurana/RaftLab/internal/storage/wal"
)

type Node struct {
	mu            sync.RWMutex
	rpcClient     *rpc.Client
	config        *config.Config
	electionTimer *electionTimer
	role          Role
	heartbeat     *heartbeatManager
	metadata      *metadata.Store
	persistent    PersistentState
	volatile      VolatileState
	pending       map[uint64]chan error
	wal           *wal.WAL
	snapshotStore *snapshot.FileStore

	stateMachine *statemachine.KVStore

	stopCh chan struct{}
}

func (n *Node) VolatileState() {
	panic("unimplemented")
}
