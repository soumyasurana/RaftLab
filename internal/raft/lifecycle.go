package raft

import (
	"os"
	"time"

	"github.com/soumyasurana/RaftLab/internal/chaos"
	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/internal/rpc"
	"github.com/soumyasurana/RaftLab/internal/snapshot"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/internal/storage/metadata"
	"github.com/soumyasurana/RaftLab/internal/storage/wal"
)

func New(cfg *config.Config) (*Node, error) {
	return NewWithTransport(cfg, nil)
}

func NewWithTransport(
	cfg *config.Config,
	transport rpc.Transport,
) (*Node, error) {
	walPath := envOrDefault("RAFT_WAL_PATH", cfg.Node.DataDir+"/raft.wal")
	metadataPath := envOrDefault("RAFT_METADATA_PATH", cfg.Node.DataDir+"/metadata.json")
	snapshotDir := envOrDefault("RAFT_SNAPSHOT_DIR", cfg.Node.DataDir)

	metaStore, err := metadata.Open(
		metadataPath,
	)
	if err != nil {
		return nil, err
	}

	persistentState, err := metaStore.Load()
	if err != nil {
		return nil, err
	}

	log, err := wal.Open(
		walPath,
	)
	if err != nil {
		return nil, err
	}

	// Read WAL to ensure it's not corrupted
	if _, err := log.ReadAll(); err != nil {
		_ = log.Close()
		return nil, err
	}

	snapshotStore := snapshot.NewFileStore(snapshotDir)
	snap, exists, err := snapshotStore.Load()
	if err != nil {
		return nil, err
	}

	stateMachine := statemachine.New()
	var lastIncludedIndex, lastIncludedTerm uint64

	if exists {
		if err := stateMachine.Restore(snap.Data); err != nil {
			return nil, err
		}
		lastIncludedIndex = snap.LastIncludedIndex
		lastIncludedTerm = snap.LastIncludedTerm
	}

	node := &Node{
		config:    cfg,
		startedAt: time.Now().UTC(),

		persistent: PersistentState{
			CurrentTerm: uint64(persistentState.CurrentTerm),
			VotedFor:    persistentState.VotedFor,
		},
		volatile: VolatileState{
			CommitIndex:       lastIncludedIndex,
			LastApplied:       lastIncludedIndex,
			LastIncludedIndex: lastIncludedIndex,
			LastIncludedTerm:  lastIncludedTerm,
		},
		pending:       make(map[uint64]chan error),
		wal:           log,
		metadata:      metaStore,
		snapshotStore: snapshotStore,

		stateMachine: stateMachine,
	}

	// Replay committed entries
	if err := node.applyCommittedEntries(); err != nil {
		_ = log.Close()
		return nil, err
	}

	// Become Follower
	node.role = Follower

	// Join the cluster
	baseTransport := transport
	if baseTransport == nil {
		baseTransport = rpc.NewClient(string(cfg.Node.ID))
	}

	node.chaosController = chaos.NewController(chaos.Config{})
	rpcClient := node.chaosController.Wrap(cfg.Node.ID, baseTransport)

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

	node.rpcClient = rpcClient

	node.heartbeat = newHeartbeatManager(
		cfg.Node.HeartbeatTimeout,
	)

	node.electionTimer = newElectionTimer(
		cfg.Node.ElectionTimeout,
		cfg.Node.ElectionTimeout*2,
	)

	node.stopCh = make(chan struct{})

	return node, nil
}

func (n *Node) Start() {
	go n.electionTimer.run(n.handleElectionTimeout)
}

func (n *Node) Stop() error {
	n.electionTimer.stop()

	close(n.stopCh)

	var rpcErr error
	if n.rpcClient != nil {
		rpcErr = n.rpcClient.Close()
	}
	walErr := n.wal.Close()

	if rpcErr != nil {
		return rpcErr
	}

	return walErr
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
