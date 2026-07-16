package raft

import (
	"sync"
	"time"

	"github.com/soumyasurana/RaftLab/internal/chaos"
	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/internal/rpc"
	"github.com/soumyasurana/RaftLab/internal/snapshot"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/internal/storage/metadata"
	"github.com/soumyasurana/RaftLab/internal/storage/wal"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

type Node struct {
	mu              sync.RWMutex
	rpcClient       rpc.Transport
	config          *config.Config
	electionTimer   *electionTimer
	role            Role
	heartbeat       *heartbeatManager
	metadata        *metadata.Store
	persistent      PersistentState
	volatile        VolatileState
	leaderID        types.NodeID
	startedAt       time.Time
	metrics         runtimeMetrics
	pending         map[uint64]chan error
	wal             *wal.WAL
	snapshotStore   *snapshot.FileStore
	chaosController *chaos.Controller

	stateMachine *statemachine.KVStore

	stopCh chan struct{}
}
