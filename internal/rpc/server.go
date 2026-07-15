package rpc

import (
	"context"
	"fmt"
	"net"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"google.golang.org/grpc"
)

/*
RaftHandler defines the Raft operations required by the gRPC server.

The RPC package depends on this interface rather than a concrete Raft node.
This prevents the transport layer from owning consensus logic and makes the
server easier to test.
*/
type RaftHandler interface {
	HandleRequestVote(
		ctx context.Context,
		req *pb.RequestVoteRequest,
	) (*pb.RequestVoteResponse, error)

	HandleAppendEntries(
		ctx context.Context,
		req *pb.AppendEntriesRequest,
	) (*pb.AppendEntriesResponse, error)

	HandleInstallSnapshot(
		ctx context.Context,
		req *pb.InstallSnapshotRequest,
	) (*pb.InstallSnapshotResponse, error)
}

// Server exposes the Raft consensus protocol over gRPC.
type Server struct {
	pb.UnimplementedRaftServiceServer

	address    string
	handler    RaftHandler
	grpcServer *grpc.Server
}

// NewServer creates a Raft gRPC server.
func NewServer(address string, handler RaftHandler) *Server {
	server := &Server{
		address: address,
		handler: handler,
	}

	server.grpcServer = grpc.NewServer()
	pb.RegisterRaftServiceServer(server.grpcServer, server)

	return server
}

/*
Start starts listening for Raft RPC requests.
Start blocks until the server stops or returns an error, so call it from a
goroutine when the node has other lifecycle work to perform.
*/
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf(
			"listen on Raft RPC address %s: %w",
			s.address,
			err,
		)
	}

	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("serve Raft RPC requests: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

// RequestVote handles an incoming RequestVote RPC.
func (s *Server) RequestVote(
	ctx context.Context,
	req *pb.RequestVoteRequest,
) (*pb.RequestVoteResponse, error) {
	return s.handler.HandleRequestVote(ctx, req)
}

// AppendEntries handles an incoming AppendEntries RPC.
func (s *Server) AppendEntries(
	ctx context.Context,
	req *pb.AppendEntriesRequest,
) (*pb.AppendEntriesResponse, error) {
	return s.handler.HandleAppendEntries(ctx, req)
}

// InstallSnapshot handles an incoming InstallSnapshot RPC.
func (s *Server) InstallSnapshot(
	ctx context.Context,
	req *pb.InstallSnapshotRequest,
) (*pb.InstallSnapshotResponse, error) {
	return s.handler.HandleInstallSnapshot(ctx, req)
}
