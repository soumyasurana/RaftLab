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

	if nextIndex <= n.volatile.LastIncludedIndex {
		n.mu.RUnlock()
		n.sendInstallSnapshot(peerID)
		return
	}

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

	n.mu.Lock()
	n.metrics.AppendEntriesSent++
	n.mu.Unlock()

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
		n.mu.Lock()
		n.metrics.RPCFailures++
		n.mu.Unlock()
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

	nodeID := types.NodeID(peerID)

	if response.Success {

		n.volatile.NextIndex[nodeID] += uint64(replicated)

		if replicated > 0 {
			n.volatile.MatchIndex[nodeID] =
				n.volatile.NextIndex[nodeID] - 1

			// Check whether a quorum has replicated a new entry.
			n.advanceCommitIndex()
		}

		return
	}

	// Replication failed.
	// Decrement NextIndex so the next AppendEntries retries from an earlier log position.
	if n.volatile.NextIndex[nodeID] > 1 {
		n.volatile.NextIndex[nodeID]--
	}
}

// sendInstallSnapshot sends the most recent snapshot to a follower.
func (n *Node) sendInstallSnapshot(peerID string) {
	snap, exists, err := n.snapshotStore.Load()
	if err != nil || !exists {
		return
	}

	n.mu.RLock()
	if n.role != Leader {
		n.mu.RUnlock()
		return
	}
	term := n.persistent.CurrentTerm
	n.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), appendEntriesTimeout*2)
	defer cancel()

	req := &pb.InstallSnapshotRequest{
		Term:              term,
		LeaderId:          string(n.config.Node.ID),
		LastIncludedIndex: snap.LastIncludedIndex,
		LastIncludedTerm:  snap.LastIncludedTerm,
		Data:              snap.Data,
	}

	resp, err := n.rpcClient.InstallSnapshot(ctx, peerID, req)
	if err != nil {
		n.mu.Lock()
		n.metrics.RPCFailures++
		n.mu.Unlock()
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if resp.Term > n.persistent.CurrentTerm {
		n.becomeFollowerLocked(resp.Term)
		return
	}

	if n.role != Leader {
		return
	}

	nodeID := types.NodeID(peerID)
	// After successful installation, nextIndex is at least LastIncludedIndex + 1
	if snap.LastIncludedIndex+1 > n.volatile.NextIndex[nodeID] {
		n.volatile.NextIndex[nodeID] = snap.LastIncludedIndex + 1
		n.volatile.MatchIndex[nodeID] = snap.LastIncludedIndex
	}
}
