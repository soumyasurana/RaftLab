package raft

import (
	"errors"
	"os"
	"path/filepath"
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

func New(cfg *config.Config) (*Node, error) {
	return NewWithTransport(cfg, nil)
}

func NewWithTransport(
	cfg *config.Config,
	transport rpc.Transport,
) (*Node, error) {
	walPath := envOrDefault("RAFT_WAL_PATH", filepath.Join(cfg.Node.DataDir, "raft.wal"))
	metadataPath := envOrDefault("RAFT_METADATA_PATH", filepath.Join(cfg.Node.DataDir, "metadata.json"))
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

	keepWAL := false
	defer func() {
		if !keepWAL {
			_ = log.Close()
		}
	}()

	// Read WAL to ensure it's not corrupted
	if _, err := log.ReadAll(); err != nil {
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
		if err := log.TruncateBefore(lastIncludedIndex); err != nil {
			return nil, err
		}
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

	node.replicating = make(map[types.NodeID]uint64)
	keepWAL = true

	return node, nil
}

func (n *Node) Start() {
	n.startOnce.Do(func() {
		go n.electionTimer.run(n.handleElectionTimeout)
	})
}

func (n *Node) Stop() error {
	n.stopOnce.Do(func() {
		n.mu.Lock()
		n.stopped = true
		n.electionTimer.stop()
		n.heartbeat.stop()
		n.mu.Unlock()

		n.stopErr = errors.Join(n.rpcClient.Close(), n.wal.Close())
	})

	return n.stopErr
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
