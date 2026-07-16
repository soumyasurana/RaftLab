package rpc

import (
	"context"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
)

// Transport is the minimal client-side RPC contract used by Raft.
type Transport interface {
	Connect(peerID string, address string) error
	Connected(peerID string) bool
	RequestVote(
		ctx context.Context,
		peerID string,
		req *pb.RequestVoteRequest,
	) (*pb.RequestVoteResponse, error)
	AppendEntries(
		ctx context.Context,
		peerID string,
		req *pb.AppendEntriesRequest,
	) (*pb.AppendEntriesResponse, error)
	InstallSnapshot(
		ctx context.Context,
		peerID string,
		req *pb.InstallSnapshotRequest,
	) (*pb.InstallSnapshotResponse, error)
	Close() error
}
