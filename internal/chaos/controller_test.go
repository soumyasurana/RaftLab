package chaos

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/internal/rpc"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

type mockTransport struct {
	mu sync.Mutex

	connects      int
	requestVotes  int
	appendEntries int
	installSnaps  int
	closed        bool
	lastPeer      string
	lastAddress   string
}

func (m *mockTransport) Connect(peerID string, address string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connects++
	m.lastPeer = peerID
	m.lastAddress = address
	return nil
}

func (m *mockTransport) Connected(string) bool {
	return true
}

func (m *mockTransport) RequestVote(
	_ context.Context,
	peerID string,
	req *pb.RequestVoteRequest,
) (*pb.RequestVoteResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestVotes++
	m.lastPeer = peerID
	return &pb.RequestVoteResponse{Term: req.Term, VoteGranted: true}, nil
}

func (m *mockTransport) AppendEntries(
	_ context.Context,
	peerID string,
	req *pb.AppendEntriesRequest,
) (*pb.AppendEntriesResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.appendEntries++
	m.lastPeer = peerID
	return &pb.AppendEntriesResponse{Term: req.Term, Success: true}, nil
}

func (m *mockTransport) InstallSnapshot(
	_ context.Context,
	peerID string,
	req *pb.InstallSnapshotRequest,
) (*pb.InstallSnapshotResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.installSnaps++
	m.lastPeer = peerID
	return &pb.InstallSnapshotResponse{Term: req.Term}, nil
}

func (m *mockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return nil
}

func TestControllerRandomPacketLoss(t *testing.T) {
	ctrl := NewController(Config{
		Enabled:               true,
		PacketDropProbability: 0.5,
		RandomSeed:            42,
	})

	base := &mockTransport{}
	transport := ctrl.Wrap("node-1", base)

	var dropped, passed int
	for i := 0; i < 16; i++ {
		_, err := transport.AppendEntries(
			context.Background(),
			"node-2",
			&pb.AppendEntriesRequest{Term: 1},
		)
		if err != nil {
			dropped++
			continue
		}
		passed++
	}

	if dropped == 0 || passed == 0 {
		t.Fatalf("expected a mix of dropped and delivered RPCs, got dropped=%d passed=%d", dropped, passed)
	}
}

func TestControllerPartitionAndHealing(t *testing.T) {
	ctrl := NewController(Config{Enabled: true})
	ctrl.SetPartitions(Partition{
		Groups: [][]types.NodeID{
			{types.NodeID("node-1"), types.NodeID("node-2")},
			{types.NodeID("node-3")},
		},
	})

	base := &mockTransport{}
	transport := ctrl.Wrap("node-1", base)

	_, err := transport.RequestVote(
		context.Background(),
		"node-3",
		&pb.RequestVoteRequest{Term: 2},
	)
	if !errors.Is(err, ErrPartitioned) {
		t.Fatalf("expected partition error, got %v", err)
	}

	_, err = transport.RequestVote(
		context.Background(),
		"node-2",
		&pb.RequestVoteRequest{Term: 2},
	)
	if err != nil {
		t.Fatalf("expected intra-partition RPC to succeed, got %v", err)
	}

	ctrl.ClearPartitions()

	_, err = transport.RequestVote(
		context.Background(),
		"node-3",
		&pb.RequestVoteRequest{Term: 2},
	)
	if err != nil {
		t.Fatalf("expected healed network to allow RPC, got %v", err)
	}
}

func TestControllerCrashDisconnectAndRecovery(t *testing.T) {
	ctrl := NewController(Config{Enabled: true})
	base := &mockTransport{}
	transport := ctrl.Wrap("node-1", base)

	ctrl.Disconnect("node-1")
	_, err := transport.AppendEntries(context.Background(), "node-2", &pb.AppendEntriesRequest{})
	if !errors.Is(err, ErrDisconnected) {
		t.Fatalf("expected disconnected error, got %v", err)
	}

	ctrl.Reconnect("node-1")
	ctrl.Crash("node-2")
	_, err = transport.AppendEntries(context.Background(), "node-2", &pb.AppendEntriesRequest{})
	if !errors.Is(err, ErrCrashed) {
		t.Fatalf("expected crash error, got %v", err)
	}

	ctrl.Restart("node-2")
	_, err = transport.AppendEntries(context.Background(), "node-2", &pb.AppendEntriesRequest{})
	if err != nil {
		t.Fatalf("expected restarted node to allow RPCs, got %v", err)
	}
}

func TestControllerDelayedRPCs(t *testing.T) {
	ctrl := NewController(Config{
		Enabled:    true,
		MinDelay:   20 * time.Millisecond,
		MaxDelay:   20 * time.Millisecond,
		RandomSeed: 1,
	})

	base := &mockTransport{}
	transport := ctrl.Wrap("node-1", base)

	start := time.Now()
	_, err := transport.AppendEntries(
		context.Background(),
		"node-2",
		&pb.AppendEntriesRequest{Term: 1},
	)
	if err != nil {
		t.Fatalf("expected delayed append to succeed, got %v", err)
	}
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Fatalf("expected at least 20ms delay, got %s", elapsed)
	}

	start = time.Now()
	_, err = transport.RequestVote(
		context.Background(),
		"node-2",
		&pb.RequestVoteRequest{Term: 1},
	)
	if err != nil {
		t.Fatalf("expected delayed vote to succeed, got %v", err)
	}
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Fatalf("expected at least 20ms delay, got %s", elapsed)
	}
}

func TestControllerDisableAndReEnable(t *testing.T) {
	ctrl := NewController(Config{
		Enabled:               true,
		PacketDropProbability: 1.0,
		RandomSeed:            7,
	})

	base := &mockTransport{}
	transport := ctrl.Wrap("node-1", base)

	_, err := transport.InstallSnapshot(
		context.Background(),
		"node-2",
		&pb.InstallSnapshotRequest{Term: 1},
	)
	if !errors.Is(err, ErrDropped) {
		t.Fatalf("expected dropped snapshot RPC, got %v", err)
	}

	ctrl.SetEnabled(false)

	_, err = transport.InstallSnapshot(
		context.Background(),
		"node-2",
		&pb.InstallSnapshotRequest{Term: 1},
	)
	if err != nil {
		t.Fatalf("expected disabled controller to pass RPC through, got %v", err)
	}

	ctrl.SetEnabled(true)
	_, err = transport.InstallSnapshot(
		context.Background(),
		"node-2",
		&pb.InstallSnapshotRequest{Term: 1},
	)
	if !errors.Is(err, ErrDropped) {
		t.Fatalf("expected dropped snapshot RPC after re-enable, got %v", err)
	}
}

func TestControllerReportsTransportClose(t *testing.T) {
	ctrl := NewController(Config{Enabled: true})
	base := &mockTransport{}
	transport := ctrl.Wrap("node-1", base)

	if err := transport.Close(); err != nil {
		t.Fatalf("close transport: %v", err)
	}

	if !base.closed {
		t.Fatal("expected base transport to be closed")
	}
}

func TestControllerAllFaultTypesAffectRaftRPCs(t *testing.T) {
	ctrl := NewController(Config{
		Enabled:               true,
		PacketDropProbability: 0.0,
		MinDelay:              5 * time.Millisecond,
		MaxDelay:              5 * time.Millisecond,
		RandomSeed:            11,
	})
	ctrl.SetPartitions(Partition{
		Groups: [][]types.NodeID{
			{types.NodeID("node-1"), types.NodeID("node-2")},
		},
	})

	base := &mockTransport{}
	transport := ctrl.Wrap("node-1", base)

	_, err := transport.RequestVote(context.Background(), "node-2", &pb.RequestVoteRequest{Term: 3})
	if err != nil {
		t.Fatalf("expected allowed request vote, got %v", err)
	}

	_, err = transport.AppendEntries(context.Background(), "node-2", &pb.AppendEntriesRequest{Term: 3})
	if err != nil {
		t.Fatalf("expected allowed append entries, got %v", err)
	}

	_, err = transport.InstallSnapshot(context.Background(), "node-2", &pb.InstallSnapshotRequest{Term: 3})
	if err != nil {
		t.Fatalf("expected allowed install snapshot, got %v", err)
	}
}

var _ rpc.Transport = (*mockTransport)(nil)
