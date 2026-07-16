package api

import (
	"context"
	"time"
)

// NodeInfo exposes the read-only management surface required by the API.
type NodeInfo interface {
	Health(ctx context.Context) (HealthResponse, error)
	Status(ctx context.Context) (StatusResponse, error)
	Peers(ctx context.Context) ([]PeerResponse, error)
	StateMachineSnapshot(ctx context.Context) (map[string]string, error)
	Metrics(ctx context.Context) (MetricsResponse, error)
}

// Manager extends NodeInfo with control-plane actions.
type Manager interface {
	NodeInfo
	TriggerSnapshot(ctx context.Context) (SnapshotResponse, error)
	EnableChaos(ctx context.Context) error
	DisableChaos(ctx context.Context) error
	ResetChaos(ctx context.Context) error
	ChaosStatus(ctx context.Context) (ChaosStatusResponse, error)
	SetChaosLatency(ctx context.Context, minDelay time.Duration, maxDelay time.Duration) error
	SetChaosPacketLoss(ctx context.Context, probability float64) error
	SetChaosPartition(ctx context.Context, groups [][]string) error
	CrashNode(ctx context.Context, nodeID string) error
	RestartNode(ctx context.Context, nodeID string) error
	DisconnectNode(ctx context.Context, nodeID string) error
	ReconnectNode(ctx context.Context, nodeID string) error
}

// HealthResponse describes the node's liveness and identity.
type HealthResponse struct {
	NodeID       string        `json:"nodeId"`
	Role         string        `json:"role"`
	CurrentTerm  uint64        `json:"currentTerm"`
	LeaderID     string        `json:"leaderId,omitempty"`
	Uptime       time.Duration `json:"uptime"`
	BuildVersion string        `json:"buildVersion"`
}

// StatusResponse describes Raft state and log/snapshot metadata.
type StatusResponse struct {
	Role              string         `json:"role"`
	CurrentTerm       uint64         `json:"currentTerm"`
	VotedFor          string         `json:"votedFor,omitempty"`
	CommitIndex       uint64         `json:"commitIndex"`
	LastApplied       uint64         `json:"lastApplied"`
	LastIncludedIndex uint64         `json:"lastIncludedIndex"`
	LastIncludedTerm  uint64         `json:"lastIncludedTerm"`
	LogLength         uint64         `json:"logLength"`
	Snapshot          SnapshotStatus `json:"snapshot"`
}

// SnapshotStatus describes whether a snapshot is available.
type SnapshotStatus struct {
	Available         bool   `json:"available"`
	LastIncludedIndex uint64 `json:"lastIncludedIndex"`
	LastIncludedTerm  uint64 `json:"lastIncludedTerm"`
	SizeBytes         uint64 `json:"sizeBytes"`
}

// PeerResponse describes a cluster peer from the local node's perspective.
type PeerResponse struct {
	PeerID          string `json:"peerId"`
	Address         string `json:"address"`
	ConnectionState string `json:"connectionState"`
	NextIndex       uint64 `json:"nextIndex"`
	MatchIndex      uint64 `json:"matchIndex"`
}

// MetricsResponse aggregates runtime counters for observability.
type MetricsResponse struct {
	ElectionsWon       uint64        `json:"electionsWon"`
	ElectionsLost      uint64        `json:"electionsLost"`
	VotesGranted       uint64        `json:"votesGranted"`
	VotesRejected      uint64        `json:"votesRejected"`
	AppendEntriesSent  uint64        `json:"appendEntriesSent"`
	AppendEntriesRecv  uint64        `json:"appendEntriesReceived"`
	SnapshotsCreated   uint64        `json:"snapshotsCreated"`
	SnapshotsInstalled uint64        `json:"snapshotsInstalled"`
	RPCFailures        uint64        `json:"rpcFailures"`
	LeaderChanges      uint64        `json:"leaderChanges"`
	Uptime             time.Duration `json:"uptime"`
}

// SnapshotResponse is returned after a manual snapshot request.
type SnapshotResponse struct {
	SnapshotIndex uint64 `json:"snapshotIndex"`
	SnapshotTerm  uint64 `json:"snapshotTerm"`
	Success       bool   `json:"success"`
}

// ChaosNodeState describes node-level chaos state.
type ChaosNodeState struct {
	Disconnected bool `json:"disconnected"`
	Crashed      bool `json:"crashed"`
}

// ChaosPartition describes a topology split injected into the cluster.
type ChaosPartition struct {
	Groups [][]string `json:"groups"`
}

// ChaosStatusResponse reports the current fault-injection configuration.
type ChaosStatusResponse struct {
	Enabled               bool                      `json:"enabled"`
	PacketDropProbability float64                   `json:"packetDropProbability"`
	MinDelayMs            int64                     `json:"minDelayMs"`
	MaxDelayMs            int64                     `json:"maxDelayMs"`
	Partitions            []ChaosPartition          `json:"partitions"`
	Nodes                 map[string]ChaosNodeState `json:"nodes"`
}

// ChaosLatencyRequest updates the simulated latency range.
type ChaosLatencyRequest struct {
	MinDelayMs int64 `json:"minDelayMs"`
	MaxDelayMs int64 `json:"maxDelayMs"`
}

// ChaosPacketLossRequest updates the packet loss probability.
type ChaosPacketLossRequest struct {
	Probability float64 `json:"probability"`
}

// ChaosPartitionRequest updates the active network partitions.
type ChaosPartitionRequest struct {
	Groups [][]string `json:"groups"`
}

// ChaosNodeRequest targets a specific node for failure injection.
type ChaosNodeRequest struct {
	NodeID string `json:"nodeId"`
}

// ActionResponse is a generic success envelope for control-plane actions.
type ActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
