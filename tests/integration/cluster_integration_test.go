package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/internal/raft"
	"github.com/soumyasurana/RaftLab/internal/rpc"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

type testCluster struct {
	nodes     map[types.NodeID]*raft.Node
	servers   map[types.NodeID]*rpc.Server
	listeners map[types.NodeID]net.Listener
	tempDir   string
}

func newTestCluster(t *testing.T, count int) *testCluster {
	t.Helper()

	dir := t.TempDir()
	cluster := &testCluster{
		nodes:   make(map[types.NodeID]*raft.Node),
		servers: make(map[types.NodeID]*rpc.Server),
		tempDir: dir,
	}

	nodeIDs := make([]types.NodeID, count)
	addresses := make(map[types.NodeID]string)

	// Pick unused ports by opening and closing listeners
	for i := 0; i < count; i++ {
		nodeIDs[i] = types.NodeID(fmt.Sprintf("node-%d", i+1))
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to get free port for node %s: %v", nodeIDs[i], err)
		}
		addresses[nodeIDs[i]] = lis.Addr().String()
		_ = lis.Close()
	}

	for _, id := range nodeIDs {
		var peers []types.Peer
		for _, peerID := range nodeIDs {
			if peerID != id {
				peers = append(peers, types.Peer{
					ID:      peerID,
					Address: addresses[peerID],
				})
			}
		}

		cfg := &config.Config{
			Node: types.NodeConfig{
				ID:                id,
				Address:           addresses[id],
				DataDir:           filepath.Join(dir, string(id)),
				ElectionTimeout:   150 * time.Millisecond,
				HeartbeatTimeout:  50 * time.Millisecond,
				SnapshotThreshold: 5,
				Peers:             peers,
			},
		}

		if err := os.MkdirAll(cfg.Node.DataDir, 0755); err != nil {
			t.Fatalf("failed to create data dir: %v", err)
		}

		node, err := raft.New(cfg)
		if err != nil {
			t.Fatalf("failed to create raft node %s: %v", id, err)
		}

		rpcServer := rpc.NewServer(addresses[id], node)
		cluster.nodes[id] = node
		cluster.servers[id] = rpcServer

		go func(s *rpc.Server) {
			_ = s.Start()
		}(rpcServer)

		node.Start()
	}

	return cluster
}

func (c *testCluster) stop() {
	for _, node := range c.nodes {
		_ = node.Stop()
	}
	for _, server := range c.servers {
		server.Stop()
	}
}

func (c *testCluster) findLeader(t *testing.T, timeout time.Duration) *raft.Node {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, node := range c.nodes {
			health, err := node.Health(context.Background())
			if err == nil && health.Role == "Leader" {
				return node
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

func TestClusterLeaderElection(t *testing.T) {
	cluster := newTestCluster(t, 3)
	defer cluster.stop()

	leader := cluster.findLeader(t, 3*time.Second)
	if leader == nil {
		t.Fatalf("cluster failed to elect a leader within 3s")
	}

	health, err := leader.Health(context.Background())
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	if health.Role != "Leader" {
		t.Fatalf("expected role Leader, got %s", health.Role)
	}
}

func TestClusterLogReplicationAndStateConsistency(t *testing.T) {
	cluster := newTestCluster(t, 3)
	defer cluster.stop()

	leader := cluster.findLeader(t, 3*time.Second)
	if leader == nil {
		t.Fatalf("no leader elected")
	}

	cmd := statemachine.Command{
		Operation: statemachine.OpSet,
		Key:       "foo",
		Value:     "bar",
	}

	if err := leader.Propose(cmd); err != nil {
		t.Fatalf("failed to propose command to leader: %v", err)
	}

	// Wait for state machine application across nodes
	deadline := time.Now().Add(3 * time.Second)
	success := false

	for time.Now().Before(deadline) {
		allConsistent := true
		for _, node := range cluster.nodes {
			snap, err := node.StateMachineSnapshot(context.Background())
			if err != nil || snap["foo"] != "bar" {
				allConsistent = false
				break
			}
		}
		if allConsistent {
			success = true
			break
		}
		time.Sleep(30 * time.Millisecond)
	}

	if !success {
		t.Fatalf("state machine snapshot did not converge to foo=bar on all nodes")
	}
}

func TestClusterSnapshotAndCompaction(t *testing.T) {
	cluster := newTestCluster(t, 3)
	defer cluster.stop()

	leader := cluster.findLeader(t, 3*time.Second)
	if leader == nil {
		t.Fatalf("no leader elected")
	}

	// Propose enough entries to exceed SnapshotThreshold = 5
	for i := 1; i <= 8; i++ {
		cmd := statemachine.Command{
			Operation: statemachine.OpSet,
			Key:       fmt.Sprintf("key-%d", i),
			Value:     fmt.Sprintf("val-%d", i),
		}
		if err := leader.Propose(cmd); err != nil {
			t.Fatalf("failed to propose key-%d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Trigger manual snapshot or wait for auto snapshot
	_, err := leader.TriggerSnapshot(context.Background())
	if err != nil {
		t.Fatalf("trigger snapshot: %v", err)
	}

	status, err := leader.Status(context.Background())
	if err != nil {
		t.Fatalf("failed to query status: %v", err)
	}

	if !status.Snapshot.Available {
		t.Fatalf("expected snapshot to be available")
	}
	if status.LastIncludedIndex == 0 {
		t.Fatalf("expected LastIncludedIndex > 0 after snapshot")
	}
}

func TestClusterPartitionTolerance(t *testing.T) {
	cluster := newTestCluster(t, 3)
	defer cluster.stop()

	leader := cluster.findLeader(t, 3*time.Second)
	if leader == nil {
		t.Fatalf("no leader elected")
	}

	leaderHealth, _ := leader.Health(context.Background())
	leaderID := leaderHealth.NodeID

	// Create partition: isolate leader from the other two nodes
	var remaining []string
	for id := range cluster.nodes {
		if string(id) != leaderID {
			remaining = append(remaining, string(id))
		}
	}

	// Inject partition on all nodes
	for _, node := range cluster.nodes {
		if err := node.SetChaosPartition(context.Background(), [][]string{{leaderID}, remaining}); err != nil {
			t.Fatalf("failed to inject partition: %v", err)
		}
		if err := node.EnableChaos(context.Background()); err != nil {
			t.Fatalf("enable chaos: %v", err)
		}
	}

	// Remaining nodes should elect a new leader among themselves
	deadline := time.Now().Add(4 * time.Second)
	newLeaderElected := false
	for time.Now().Before(deadline) {
		for id, node := range cluster.nodes {
			if string(id) != leaderID {
				h, err := node.Health(context.Background())
				if err == nil && h.Role == "Leader" {
					newLeaderElected = true
					break
				}
			}
		}
		if newLeaderElected {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}

	if !newLeaderElected {
		t.Fatalf("expected remaining partition to elect a new leader")
	}

	// Heal partition
	for _, node := range cluster.nodes {
		if err := node.ResetChaos(context.Background()); err != nil {
			t.Fatalf("reset chaos: %v", err)
		}
	}
}
