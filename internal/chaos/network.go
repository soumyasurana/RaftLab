package chaos

import (
	"context"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/internal/rpc"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

type transport struct {
	controller *Controller
	ownerID    types.NodeID
	base       rpc.Transport
}

func (t *transport) Connect(peerID string, address string) error {
	return t.base.Connect(peerID, address)
}

func (t *transport) RequestVote(
	ctx context.Context,
	peerID string,
	req *pb.RequestVoteRequest,
) (*pb.RequestVoteResponse, error) {
	return t.controller.requestVote(
		ctx,
		t.ownerID,
		types.NodeID(peerID),
		req,
		t.base,
	)
}

func (t *transport) AppendEntries(
	ctx context.Context,
	peerID string,
	req *pb.AppendEntriesRequest,
) (*pb.AppendEntriesResponse, error) {
	return t.controller.appendEntries(
		ctx,
		t.ownerID,
		types.NodeID(peerID),
		req,
		t.base,
	)
}

func (t *transport) InstallSnapshot(
	ctx context.Context,
	peerID string,
	req *pb.InstallSnapshotRequest,
) (*pb.InstallSnapshotResponse, error) {
	return t.controller.installSnapshot(
		ctx,
		t.ownerID,
		types.NodeID(peerID),
		req,
		t.base,
	)
}

func (t *transport) Close() error {
	return t.base.Close()
}
