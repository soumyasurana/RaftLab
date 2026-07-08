package raft

import (
	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/internal/storage/wal"
)

func New(cfg *config.Config) (*Node, error) {
	log, err := wal.Open(cfg.Node.DataDir + "/raft.wal")
	if err != nil {
		return nil, err
	}

	node := &Node{
		config: cfg,

		role: Follower,

		wal: log,

		stateMachine: statemachine.New(),

		stopCh: make(chan struct{}),
	}

	return node, nil
}

func (n *Node) Start() {
	// Timers will be started here later.
}

func (n *Node) Stop() error {
	close(n.stopCh)
	return n.wal.Close()
}
