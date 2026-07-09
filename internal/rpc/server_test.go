package rpc

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
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

func TestRequestVoteRPC(t *testing.T) {
	address := freeAddress(t)

	server := NewServer(
		address,
		&mockRaftHandler{},
	)

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.Start()
	}()

	t.Cleanup(server.Stop)

	client := NewClient()

	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close RPC client: %v", err)
		}
	})

	if err := client.Connect("node-2", address); err != nil {
		t.Fatalf("connect to RPC server: %v", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	response, err := client.RequestVote(
		ctx,
		"node-2",
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
		t.Fatalf(
			"expected response term 4, got %d",
			response.Term,
		)
	}

	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatalf("gRPC server stopped unexpectedly: %v", err)
		}
	default:
	}
}

func TestAppendEntriesRPC(t *testing.T) {
	address := freeAddress(t)

	server := NewServer(
		address,
		&mockRaftHandler{},
	)

	go func() {
		_ = server.Start()
	}()

	t.Cleanup(server.Stop)

	client := NewClient()

	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close RPC client: %v", err)
		}
	})

	if err := client.Connect("node-2", address); err != nil {
		t.Fatalf("connect to RPC server: %v", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	response, err := client.AppendEntries(
		ctx,
		"node-2",
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
		t.Fatalf(
			"expected response term 7, got %d",
			response.Term,
		)
	}
}

func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen(
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("allocate test address: %v", err)
	}

	address := listener.Addr().String()

	if err := listener.Close(); err != nil {
		t.Fatalf("close temporary listener: %v", err)
	}

	return address
}
