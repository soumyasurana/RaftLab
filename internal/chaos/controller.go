package chaos

import (
	"context"
	"math/rand"
	"sync"
	"time"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/internal/rpc"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

type nodeState struct {
	disconnected bool
	crashed      bool
}

// Snapshot captures the controller's current config and node states.
type Snapshot struct {
	Config Config
	Nodes  map[types.NodeID]ChaosNodeState
}

// ChaosNodeState is the exported view of a node's injected fault state.
type ChaosNodeState struct {
	Disconnected bool
	Crashed      bool
}

// Controller owns global chaos state and creates node-scoped transports.
type Controller struct {
	mu sync.Mutex

	cfg Config
	rng *rand.Rand

	nodes map[types.NodeID]*nodeState
}

// NewController creates a controller with the provided config.
func NewController(cfg Config) *Controller {
	return &Controller{
		cfg:   cfg,
		rng:   rand.New(rand.NewSource(cfg.RandomSeed)),
		nodes: make(map[types.NodeID]*nodeState),
	}
}

// Wrap returns a transport that injects faults for the given owner node.
func (c *Controller) Wrap(ownerID types.NodeID, base rpc.Transport) rpc.Transport {
	c.mu.Lock()
	c.ensureNodeLocked(ownerID)
	c.mu.Unlock()

	return &transport{
		controller: c,
		ownerID:    ownerID,
		base:       base,
	}
}

// SetEnabled toggles fault injection on or off.
func (c *Controller) SetEnabled(enabled bool) {
	c.mu.Lock()
	c.cfg.Enabled = enabled
	c.mu.Unlock()
}

// UpdateConfig replaces the controller configuration while preserving runtime node state.
func (c *Controller) UpdateConfig(cfg Config) {
	c.mu.Lock()
	c.cfg = cfg
	c.mu.Unlock()
}

// SetPartitions replaces the active partition topology.
func (c *Controller) SetPartitions(partitions ...Partition) {
	c.mu.Lock()
	c.cfg.Partitions = append([]Partition(nil), partitions...)
	c.mu.Unlock()
}

// ClearPartitions removes all active partitions.
func (c *Controller) ClearPartitions() {
	c.SetPartitions()
}

// Reset clears all injected faults and restores the controller to a clean state.
func (c *Controller) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.Enabled = false
	c.cfg.PacketDropProbability = 0
	c.cfg.MinDelay = 0
	c.cfg.MaxDelay = 0
	c.cfg.Partitions = nil

	for _, state := range c.nodes {
		state.disconnected = false
		state.crashed = false
	}
}

// Snapshot returns a copy of the current controller configuration.
func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	nodes := make(map[types.NodeID]ChaosNodeState, len(c.nodes))
	for nodeID, state := range c.nodes {
		nodes[nodeID] = ChaosNodeState{
			Disconnected: state.disconnected,
			Crashed:      state.crashed,
		}
	}

	return Snapshot{
		Config: c.cfg,
		Nodes:  nodes,
	}
}

// SetLatency updates the simulated delay window.
func (c *Controller) SetLatency(minDelay, maxDelay time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.MinDelay = minDelay
	c.cfg.MaxDelay = maxDelay
}

// SetPacketDropProbability updates the packet loss rate.
func (c *Controller) SetPacketDropProbability(probability float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cfg.PacketDropProbability = probability
}

// Disconnect prevents the node from sending or receiving traffic.
func (c *Controller) Disconnect(nodeID types.NodeID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.ensureNodeLocked(nodeID)
	state.disconnected = true
}

// Reconnect restores traffic for a disconnected node.
func (c *Controller) Reconnect(nodeID types.NodeID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.ensureNodeLocked(nodeID)
	state.disconnected = false
}

// Crash marks a node as unavailable.
func (c *Controller) Crash(nodeID types.NodeID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.ensureNodeLocked(nodeID)
	state.crashed = true
}

// Restart clears the crashed state for a node.
func (c *Controller) Restart(nodeID types.NodeID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.ensureNodeLocked(nodeID)
	state.crashed = false
}

// SetPartitions replaces the partition topology using the provided groups.
func (c *Controller) SetPartitionsFromStrings(groups [][]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	partitions := make([]Partition, 0, 1)
	if len(groups) > 0 {
		converted := make([][]types.NodeID, 0, len(groups))
		for _, group := range groups {
			nodes := make([]types.NodeID, 0, len(group))
			for _, nodeID := range group {
				if nodeID == "" {
					continue
				}
				nodes = append(nodes, types.NodeID(nodeID))
			}
			if len(nodes) > 0 {
				converted = append(converted, nodes)
			}
		}
		if len(converted) > 0 {
			partitions = append(partitions, Partition{Groups: converted})
		}
	}

	c.cfg.Partitions = partitions
}

func (c *Controller) ensureNodeLocked(nodeID types.NodeID) *nodeState {
	state, ok := c.nodes[nodeID]
	if !ok {
		state = &nodeState{}
		c.nodes[nodeID] = state
	}

	return state
}

func (c *Controller) decisionLocked(ownerID, peerID types.NodeID) (time.Duration, error) {
	if !c.cfg.Enabled {
		return 0, nil
	}

	ownerState := c.ensureNodeLocked(ownerID)
	peerState := c.ensureNodeLocked(peerID)

	if ownerState.crashed || peerState.crashed {
		return 0, ErrCrashed
	}

	if ownerState.disconnected || peerState.disconnected {
		return 0, ErrDisconnected
	}

	for _, partition := range c.cfg.Partitions {
		if !partitionAllows(partition, ownerID, peerID) {
			return 0, ErrPartitioned
		}
	}

	if c.cfg.PacketDropProbability > 0 &&
		c.rng.Float64() < c.cfg.PacketDropProbability {
		return 0, ErrDropped
	}

	if c.cfg.MaxDelay <= 0 {
		if c.cfg.MinDelay > 0 {
			return c.cfg.MinDelay, nil
		}
		return 0, nil
	}

	delay := c.cfg.MaxDelay
	if c.cfg.MinDelay > 0 && c.cfg.MinDelay < c.cfg.MaxDelay {
		jitter := c.rng.Int63n(int64(c.cfg.MaxDelay-c.cfg.MinDelay) + 1)
		delay = c.cfg.MinDelay + time.Duration(jitter)
	}

	return delay, nil
}

func (c *Controller) invoke(
	ctx context.Context,
	ownerID, peerID types.NodeID,
	fn func(context.Context) (any, error),
) (any, error) {
	c.mu.Lock()
	delay, err := c.decisionLocked(ownerID, peerID)
	c.mu.Unlock()

	if err != nil {
		return nil, err
	}

	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return fn(ctx)
}

func (c *Controller) requestVote(
	ctx context.Context,
	ownerID, peerID types.NodeID,
	req *pb.RequestVoteRequest,
	base rpc.Transport,
) (*pb.RequestVoteResponse, error) {
	value, err := c.invoke(ctx, ownerID, peerID, func(ctx context.Context) (any, error) {
		return base.RequestVote(ctx, string(peerID), req)
	})
	if err != nil {
		return nil, err
	}

	return value.(*pb.RequestVoteResponse), nil
}

func (c *Controller) appendEntries(
	ctx context.Context,
	ownerID, peerID types.NodeID,
	req *pb.AppendEntriesRequest,
	base rpc.Transport,
) (*pb.AppendEntriesResponse, error) {
	value, err := c.invoke(ctx, ownerID, peerID, func(ctx context.Context) (any, error) {
		return base.AppendEntries(ctx, string(peerID), req)
	})
	if err != nil {
		return nil, err
	}

	return value.(*pb.AppendEntriesResponse), nil
}

func (c *Controller) installSnapshot(
	ctx context.Context,
	ownerID, peerID types.NodeID,
	req *pb.InstallSnapshotRequest,
	base rpc.Transport,
) (*pb.InstallSnapshotResponse, error) {
	value, err := c.invoke(ctx, ownerID, peerID, func(ctx context.Context) (any, error) {
		return base.InstallSnapshot(ctx, string(peerID), req)
	})
	if err != nil {
		return nil, err
	}

	return value.(*pb.InstallSnapshotResponse), nil
}
