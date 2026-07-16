package raft

import (
	"context"
	"fmt"
	"time"

	"github.com/soumyasurana/RaftLab/internal/api"
	"github.com/soumyasurana/RaftLab/internal/snapshot"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

type runtimeMetrics struct {
	ElectionsWon       uint64
	ElectionsLost      uint64
	VotesGranted       uint64
	VotesRejected      uint64
	AppendEntriesSent  uint64
	AppendEntriesRecv  uint64
	SnapshotsCreated   uint64
	SnapshotsInstalled uint64
	RPCFailures        uint64
	LeaderChanges      uint64
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}

// Health returns the node's liveness and role information.
func (n *Node) Health(ctx context.Context) (api.HealthResponse, error) {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return api.HealthResponse{}, err
	}

	n.mu.RLock()
	nodeID := string(n.config.Node.ID)
	role := n.role.String()
	term := n.persistent.CurrentTerm
	leaderID := string(n.leaderID)
	startedAt := n.startedAt
	n.mu.RUnlock()

	if leaderID == "" && role == Leader.String() {
		leaderID = nodeID
	}

	return api.HealthResponse{
		NodeID:      nodeID,
		Role:        role,
		CurrentTerm: term,
		LeaderID:    leaderID,
		Uptime:      time.Since(startedAt),
	}, nil
}

// Status returns the node's durable and volatile Raft state.
func (n *Node) Status(ctx context.Context) (api.StatusResponse, error) {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return api.StatusResponse{}, err
	}

	n.mu.RLock()
	status := api.StatusResponse{
		Role:              n.role.String(),
		CurrentTerm:       n.persistent.CurrentTerm,
		VotedFor:          string(n.persistent.VotedFor),
		CommitIndex:       n.volatile.CommitIndex,
		LastApplied:       n.volatile.LastApplied,
		LastIncludedIndex: n.volatile.LastIncludedIndex,
		LastIncludedTerm:  n.volatile.LastIncludedTerm,
	}
	n.mu.RUnlock()

	entries, err := n.wal.ReadAll()
	if err != nil {
		return api.StatusResponse{}, err
	}
	status.LogLength = uint64(len(entries))

	snap, exists, err := n.snapshotStore.Load()
	if err != nil {
		return api.StatusResponse{}, err
	}
	if exists {
		status.Snapshot = api.SnapshotStatus{
			Available:         true,
			LastIncludedIndex: snap.LastIncludedIndex,
			LastIncludedTerm:  snap.LastIncludedTerm,
		}
	} else {
		status.Snapshot = api.SnapshotStatus{
			Available:         false,
			LastIncludedIndex: status.LastIncludedIndex,
			LastIncludedTerm:  status.LastIncludedTerm,
		}
	}

	return status, nil
}

// Peers returns the local view of every configured peer.
func (n *Node) Peers(ctx context.Context) ([]api.PeerResponse, error) {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	n.mu.RLock()
	peers := append([]types.Peer(nil), n.config.Node.Peers...)
	nextIndex := make(map[types.NodeID]uint64, len(n.volatile.NextIndex))
	for peerID, value := range n.volatile.NextIndex {
		nextIndex[peerID] = value
	}
	matchIndex := make(map[types.NodeID]uint64, len(n.volatile.MatchIndex))
	for peerID, value := range n.volatile.MatchIndex {
		matchIndex[peerID] = value
	}
	rpcClient := n.rpcClient
	n.mu.RUnlock()

	result := make([]api.PeerResponse, 0, len(peers))
	for _, peer := range peers {
		connectionState := "unknown"
		if rpcClient != nil {
			if rpcClient.Connected(string(peer.ID)) {
				connectionState = "connected"
			} else {
				connectionState = "disconnected"
			}
		}

		result = append(result, api.PeerResponse{
			PeerID:          string(peer.ID),
			Address:         peer.Address,
			ConnectionState: connectionState,
			NextIndex:       nextIndex[peer.ID],
			MatchIndex:      matchIndex[peer.ID],
		})
	}

	return result, nil
}

// StateMachineSnapshot returns a copy of the replicated key-value store.
func (n *Node) StateMachineSnapshot(ctx context.Context) (map[string]string, error) {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return n.stateMachine.Snapshot(), nil
}

// Metrics returns the node's runtime counters.
func (n *Node) Metrics(ctx context.Context) (api.MetricsResponse, error) {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return api.MetricsResponse{}, err
	}

	n.mu.RLock()
	metrics := n.metrics
	startedAt := n.startedAt
	n.mu.RUnlock()

	return api.MetricsResponse{
		ElectionsWon:       metrics.ElectionsWon,
		ElectionsLost:      metrics.ElectionsLost,
		VotesGranted:       metrics.VotesGranted,
		VotesRejected:      metrics.VotesRejected,
		AppendEntriesSent:  metrics.AppendEntriesSent,
		AppendEntriesRecv:  metrics.AppendEntriesRecv,
		SnapshotsCreated:   metrics.SnapshotsCreated,
		SnapshotsInstalled: metrics.SnapshotsInstalled,
		RPCFailures:        metrics.RPCFailures,
		LeaderChanges:      metrics.LeaderChanges,
		Uptime:             time.Since(startedAt),
	}, nil
}

// TriggerSnapshot forces the node to compact its committed state.
func (n *Node) TriggerSnapshot(ctx context.Context) (api.SnapshotResponse, error) {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return api.SnapshotResponse{}, err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if err := n.takeSnapshot(); err != nil {
		return api.SnapshotResponse{}, err
	}

	return api.SnapshotResponse{
		SnapshotIndex: n.volatile.LastIncludedIndex,
		SnapshotTerm:  n.volatile.LastIncludedTerm,
		Success:       true,
	}, nil
}

// EnableChaos enables fault injection on the node's wrapped transport.
func (n *Node) EnableChaos(ctx context.Context) error {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	if n.chaosController == nil {
		return fmt.Errorf("chaos controller is not configured")
	}

	n.chaosController.SetEnabled(true)
	return nil
}

// DisableChaos disables fault injection on the node's wrapped transport.
func (n *Node) DisableChaos(ctx context.Context) error {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	if n.chaosController == nil {
		return fmt.Errorf("chaos controller is not configured")
	}

	n.chaosController.SetEnabled(false)
	return nil
}

// ResetChaos clears every injected fault and returns the controller to a clean state.
func (n *Node) ResetChaos(ctx context.Context) error {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	if n.chaosController == nil {
		return fmt.Errorf("chaos controller is not configured")
	}

	n.chaosController.Reset()
	return nil
}

// SnapshotInfo returns the current snapshot metadata.
func (n *Node) SnapshotInfo() (snapshot.Snapshot, bool, error) {
	return n.snapshotStore.Load()
}
