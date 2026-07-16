package raft

import (
	"context"
	"fmt"
	"time"

	"github.com/soumyasurana/RaftLab/internal/api"
	"github.com/soumyasurana/RaftLab/internal/chaos"
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
		sizeBytes, _, err := n.snapshotStore.Size()
		if err != nil {
			return api.StatusResponse{}, err
		}
		status.Snapshot = api.SnapshotStatus{
			Available:         true,
			LastIncludedIndex: snap.LastIncludedIndex,
			LastIncludedTerm:  snap.LastIncludedTerm,
			SizeBytes:         sizeBytes,
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

// ChaosStatus returns the controller state used by the dashboard.
func (n *Node) ChaosStatus(ctx context.Context) (api.ChaosStatusResponse, error) {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return api.ChaosStatusResponse{}, err
	}

	if n.chaosController == nil {
		return api.ChaosStatusResponse{}, fmt.Errorf("chaos controller is not configured")
	}

	snapshot := n.chaosController.Snapshot()
	result := api.ChaosStatusResponse{
		Enabled:               snapshot.Config.Enabled,
		PacketDropProbability: snapshot.Config.PacketDropProbability,
		MinDelayMs:            int64(snapshot.Config.MinDelay / time.Millisecond),
		MaxDelayMs:            int64(snapshot.Config.MaxDelay / time.Millisecond),
		Partitions:            make([]api.ChaosPartition, 0, len(snapshot.Config.Partitions)),
		Nodes:                 make(map[string]api.ChaosNodeState, len(snapshot.Nodes)),
	}

	for _, partition := range snapshot.Config.Partitions {
		groupStrings := make([][]string, 0, len(partition.Groups))
		for _, group := range partition.Groups {
			nodes := make([]string, 0, len(group))
			for _, nodeID := range group {
				nodes = append(nodes, string(nodeID))
			}
			groupStrings = append(groupStrings, nodes)
		}
		result.Partitions = append(result.Partitions, api.ChaosPartition{Groups: groupStrings})
	}

	for nodeID, state := range snapshot.Nodes {
		result.Nodes[string(nodeID)] = api.ChaosNodeState{
			Disconnected: state.Disconnected,
			Crashed:      state.Crashed,
		}
	}

	return result, nil
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

// SetChaosLatency updates the simulated delay window.
func (n *Node) SetChaosLatency(ctx context.Context, minDelay time.Duration, maxDelay time.Duration) error {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	if n.chaosController == nil {
		return fmt.Errorf("chaos controller is not configured")
	}

	n.chaosController.SetLatency(minDelay, maxDelay)
	return nil
}

// SetChaosPacketLoss updates the simulated packet loss rate.
func (n *Node) SetChaosPacketLoss(ctx context.Context, probability float64) error {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	if n.chaosController == nil {
		return fmt.Errorf("chaos controller is not configured")
	}

	n.chaosController.SetPacketDropProbability(probability)
	return nil
}

// SetChaosPartition updates the injected topology split.
func (n *Node) SetChaosPartition(ctx context.Context, groups [][]string) error {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	if n.chaosController == nil {
		return fmt.Errorf("chaos controller is not configured")
	}

	partitions := make([]chaos.Partition, 0, len(groups))
	for _, group := range groups {
		nodes := make([]types.NodeID, 0, len(group))
		for _, nodeID := range group {
			if nodeID == "" {
				continue
			}
			nodes = append(nodes, types.NodeID(nodeID))
		}
		if len(nodes) > 0 {
			partitions = append(partitions, chaos.Partition{Groups: [][]types.NodeID{nodes}})
		}
	}

	n.chaosController.SetPartitions(partitions...)
	return nil
}

// CrashNode marks a node as crashed in the chaos controller.
func (n *Node) CrashNode(ctx context.Context, nodeID string) error {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	if n.chaosController == nil {
		return fmt.Errorf("chaos controller is not configured")
	}

	n.chaosController.Crash(types.NodeID(nodeID))
	return nil
}

// RestartNode clears the crash state for a node.
func (n *Node) RestartNode(ctx context.Context, nodeID string) error {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	if n.chaosController == nil {
		return fmt.Errorf("chaos controller is not configured")
	}

	n.chaosController.Restart(types.NodeID(nodeID))
	return nil
}

// DisconnectNode isolates a node from the cluster.
func (n *Node) DisconnectNode(ctx context.Context, nodeID string) error {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	if n.chaosController == nil {
		return fmt.Errorf("chaos controller is not configured")
	}

	n.chaosController.Disconnect(types.NodeID(nodeID))
	return nil
}

// ReconnectNode restores a disconnected node.
func (n *Node) ReconnectNode(ctx context.Context, nodeID string) error {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	if n.chaosController == nil {
		return fmt.Errorf("chaos controller is not configured")
	}

	n.chaosController.Reconnect(types.NodeID(nodeID))
	return nil
}

// SnapshotInfo returns the current snapshot metadata.
func (n *Node) SnapshotInfo() (snapshot.Snapshot, bool, error) {
	return n.snapshotStore.Load()
}
