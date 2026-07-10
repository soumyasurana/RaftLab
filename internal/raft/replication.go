package raft

import (
	"context"
	"time"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
)

const appendEntriesTimeout = 200 * time.Millisecond

func (n *Node) replicateToPeer(peerID string) {

	n.mu.RLock()

	if n.role != Leader {
		n.mu.RUnlock()
		return
	}

	term := n.persistent.CurrentTerm
	leaderCommit := n.volatile.CommitIndex

	n.mu.RUnlock()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		appendEntriesTimeout,
	)
	defer cancel()

	response, err := n.rpcClient.AppendEntries(
		ctx,
		peerID,
		&pb.AppendEntriesRequest{
			Term:         term,
			LeaderId:     string(n.config.Node.ID),
			LeaderCommit: leaderCommit,
		},
	)

	if err != nil {
		return
	}

	n.handleAppendEntriesResponse(response)
}

// handleAppendEntriesResponse updates leader replication state.
func (n *Node) handleAppendEntriesResponse(
	response *pb.AppendEntriesResponse,
) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if response.Term > n.persistent.CurrentTerm {
		n.becomeFollowerLocked(response.Term)
	}
}
