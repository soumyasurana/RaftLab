package raft

import (
	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/internal/rpc"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/internal/storage/wal"
)

func New(cfg *config.Config) (*Node, error) {
	log, err := wal.Open(cfg.Node.DataDir + "/raft.wal")
	if err != nil {
		return nil, err
	}

	rpcClient := rpc.NewClient()

	for _, peer := range cfg.Node.Peers {
		if err := rpcClient.Connect(
			string(peer.ID),
			peer.Address,
		); err != nil {
			_ = rpcClient.Close()
			_ = log.Close()

			return nil, err
		}
	}

	node := &Node{
		config: cfg,

		role: Follower,

		wal: log,

		stateMachine: statemachine.New(),

		rpcClient: rpcClient,

		electionTimer: newElectionTimer(
			cfg.Node.ElectionTimeout,
			cfg.Node.ElectionTimeout*2,
		),

		stopCh: make(chan struct{}),
	}

	return node, nil
}

func (n *Node) Start() {
	go n.electionTimer.run(n.handleElectionTimeout)
}

func (n *Node) Stop() error {
	n.electionTimer.stop()

	close(n.stopCh)

	rpcErr := n.rpcClient.Close()
	walErr := n.wal.Close()

	if rpcErr != nil {
		return rpcErr
	}

	return walErr
}
