package raft

import (
	"context"
	"log"
	"time"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

const appendEntriesTimeout = 200 * time.Millisecond

func (n *Node) replicateToPeers(term uint64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.stopped || n.role != Leader || n.persistent.CurrentTerm != term {
		return
	}
	for _, peer := range n.config.Node.Peers {
		if inFlightTerm, ok := n.replicating[peer.ID]; ok && inFlightTerm == term {
			continue
		}
		n.replicating[peer.ID] = term
		go n.replicateToPeer(peer.ID, term)
	}
}

func (n *Node) replicateToPeer(peerID types.NodeID, term uint64) {
	defer func() {
		n.mu.Lock()
		if n.replicating[peerID] == term {
			delete(n.replicating, peerID)
		}
		n.mu.Unlock()
	}()

	n.mu.RLock()
	if n.stopped || n.role != Leader || n.persistent.CurrentTerm != term {
		n.mu.RUnlock()
		return
	}

	nextIndex := n.volatile.NextIndex[peerID]
	lastIncludedIndex := n.volatile.LastIncludedIndex
	leaderCommit := n.volatile.CommitIndex
	if nextIndex <= lastIncludedIndex {
		n.mu.RUnlock()
		n.sendInstallSnapshot(string(peerID), term)
		return
	}

	prevIndex := nextIndex - 1
	var prevTerm uint64
	if prevIndex == lastIncludedIndex {
		prevTerm = n.volatile.LastIncludedTerm
	} else {
		entry, ok, err := n.wal.EntryAt(prevIndex)
		if err != nil || !ok {
			n.mu.RUnlock()
			return
		}
		prevTerm = uint64(entry.Term)
	}

	logEntries, err := n.wal.EntriesFrom(nextIndex)
	n.mu.RUnlock()
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
		string(peerID),
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
		string(peerID),
		term,
		response,
		prevIndex,
		len(entries),
	)
}

// handleAppendEntriesResponse updates leader replication state.
func (n *Node) handleAppendEntriesResponse(
	peerID string,
	term uint64,
	response *pb.AppendEntriesResponse,
	prevIndex uint64,
	replicated int,
) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.stopped {
		return
	}
	if response.Term > n.persistent.CurrentTerm {
		n.stepDownAfterResponse(response.Term)
		return
	}

	if n.role != Leader || n.persistent.CurrentTerm != term || response.Term != term {
		return
	}

	nodeID := types.NodeID(peerID)

	if response.Success {
		matchIndex := prevIndex + uint64(replicated)
		if matchIndex > n.volatile.MatchIndex[nodeID] {
			n.volatile.MatchIndex[nodeID] = matchIndex
		}
		if matchIndex+1 > n.volatile.NextIndex[nodeID] {
			n.volatile.NextIndex[nodeID] = matchIndex + 1
		}

		if replicated > 0 {
			// Check whether a quorum has replicated a new entry.
			n.advanceCommitIndex()
		}

		return
	}

	// Replication failed.
	// Decrement NextIndex so the next AppendEntries retries from an earlier log position.
	if n.volatile.NextIndex[nodeID] == prevIndex+1 && prevIndex > n.volatile.MatchIndex[nodeID] {
		n.volatile.NextIndex[nodeID]--
	}
}

// sendInstallSnapshot sends the most recent snapshot to a follower.
func (n *Node) sendInstallSnapshot(peerID string, term uint64) {
	snap, exists, err := n.snapshotStore.Load()
	if err != nil || !exists {
		return
	}

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

	if n.stopped {
		return
	}
	if resp.Term > n.persistent.CurrentTerm {
		n.stepDownAfterResponse(resp.Term)
		return
	}

	if n.role != Leader || n.persistent.CurrentTerm != term || resp.Term != term {
		return
	}

	nodeID := types.NodeID(peerID)
	// After successful installation, nextIndex is at least LastIncludedIndex + 1
	if snap.LastIncludedIndex+1 > n.volatile.NextIndex[nodeID] {
		n.volatile.NextIndex[nodeID] = snap.LastIncludedIndex + 1
		n.volatile.MatchIndex[nodeID] = snap.LastIncludedIndex
	}
}

// stepDownAfterResponse handles a newer term observed by the leader.
// The caller must hold n.mu.
func (n *Node) stepDownAfterResponse(term uint64) {
	if err := n.becomeFollowerLocked(term); err != nil {
		log.Printf("persist newer term: %v", err)
		return
	}
	n.electionTimer.reset()
}
