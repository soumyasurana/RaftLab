package raft

import (
	"context"
	"time"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

const appendEntriesTimeout = 200 * time.Millisecond

func (n *Node) replicateToPeer(peerID string) {

	n.mu.RLock()

	if n.role != Leader {
		n.mu.RUnlock()
		return
	}

	nextIndex := n.volatile.NextIndex[types.NodeID(peerID)]

	term := n.persistent.CurrentTerm
	leaderCommit := n.volatile.CommitIndex

	n.mu.RUnlock()

	var (
		prevIndex uint64
		prevTerm  uint64
	)

	if nextIndex > 1 {

		entry, ok, err := n.wal.EntryAt(nextIndex - 1)
		if err != nil {
			return
		}

		if ok {
			prevIndex = uint64(entry.Index)
			prevTerm = uint64(entry.Term)
		}
	}

	logEntries, err := n.wal.EntriesFrom(nextIndex)
	if err != nil {
		return
	}

	entries := make([]*pb.LogEntry, 0, len(logEntries))

	for _, entry := range logEntries {

		entries = append(entries, &pb.LogEntry{
			Index:   uint64(entry.Index),
			Term:    uint64(entry.Term),
			Command: entry.Command,
		})
	}

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
			PrevLogIndex: prevIndex,
			PrevLogTerm:  prevTerm,
			Entries:      entries,
			LeaderCommit: leaderCommit,
		},
	)

	if err != nil {
		return
	}

	n.handleAppendEntriesResponse(
		peerID,
		response,
		len(entries),
	)
}

// handleAppendEntriesResponse updates leader replication state.
func (n *Node) handleAppendEntriesResponse(
	peerID string,
	response *pb.AppendEntriesResponse,
	replicated int,
) {

	n.mu.Lock()
	defer n.mu.Unlock()

	if response.Term > n.persistent.CurrentTerm {
		n.becomeFollowerLocked(response.Term)
		return
	}

	if n.role != Leader {
		return
	}

	if response.Success {

		n.volatile.NextIndex[types.NodeID(peerID)] += uint64(replicated)

		if replicated > 0 {
			n.volatile.MatchIndex[types.NodeID(peerID)] =
				n.volatile.NextIndex[types.NodeID(peerID)] - 1
		}

		return
	}

	if n.volatile.NextIndex[types.NodeID(peerID)] > 1 {
		n.volatile.NextIndex[types.NodeID(peerID)]--
	}
}
