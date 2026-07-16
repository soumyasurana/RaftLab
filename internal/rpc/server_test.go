package rpc

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type mockRaftHandler struct{}

func (m *mockRaftHandler) HandleRequestVote(
	_ context.Context,
	req *pb.RequestVoteRequest,
) (*pb.RequestVoteResponse, error) {
	return &pb.RequestVoteResponse{
		Term:        req.Term,
		VoteGranted: true,
	}, nil
}

func (m *mockRaftHandler) HandleAppendEntries(
	_ context.Context,
	req *pb.AppendEntriesRequest,
) (*pb.AppendEntriesResponse, error) {
	return &pb.AppendEntriesResponse{
		Term:    req.Term,
		Success: true,
	}, nil
}

func (m *mockRaftHandler) HandleInstallSnapshot(
	_ context.Context,
	req *pb.InstallSnapshotRequest,
) (*pb.InstallSnapshotResponse, error) {
	return &pb.InstallSnapshotResponse{
		Term: req.Term,
	}, nil
}

func TestRequestVoteRPC(t *testing.T) {
	client, cleanup := newBufconnClient(t, &mockRaftHandler{})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := client.RequestVote(
		ctx,
		&pb.RequestVoteRequest{
			Term:         4,
			CandidateId:  "node-1",
			LastLogIndex: 10,
			LastLogTerm:  3,
		},
	)
	if err != nil {
		t.Fatalf("send RequestVote RPC: %v", err)
	}

	if !response.VoteGranted {
		t.Fatal("expected vote to be granted")
	}

	if response.Term != 4 {
		t.Fatalf("expected response term 4, got %d", response.Term)
	}
}

func TestAppendEntriesRPC(t *testing.T) {
	client, cleanup := newBufconnClient(t, &mockRaftHandler{})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := client.AppendEntries(
		ctx,
		&pb.AppendEntriesRequest{
			Term:         7,
			LeaderId:     "node-1",
			PrevLogIndex: 5,
			PrevLogTerm:  6,
			LeaderCommit: 4,
		},
	)
	if err != nil {
		t.Fatalf("send AppendEntries RPC: %v", err)
	}

	if !response.Success {
		t.Fatal("expected AppendEntries to succeed")
	}

	if response.Term != 7 {
		t.Fatalf("expected response term 7, got %d", response.Term)
	}
}

func newBufconnClient(
	t *testing.T,
	handler RaftHandler,
) (pb.RaftServiceClient, func()) {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	server := NewServer("bufconn", handler)

	go func() {
		_ = server.grpcServer.Serve(lis)
	}()

	conn, err := grpc.DialContext(
		context.Background(),
		"bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn server: %v", err)
	}

	client := pb.NewRaftServiceClient(conn)

	cleanup := func() {
		server.Stop()
		_ = conn.Close()
	}

	return client, cleanup
}
