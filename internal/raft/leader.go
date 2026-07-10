package raft

import "github.com/soumyasurana/RaftLab/pkg/types"

// initializeLeaderState prepares leader-only replication state.
func (n *Node) initializeLeaderState(lastLogIndex uint64) {
	n.volatile.NextIndex = make(map[types.NodeID]uint64)
	n.volatile.MatchIndex = make(map[types.NodeID]uint64)

	for _, peer := range n.config.Node.Peers {
		n.volatile.NextIndex[peer.ID] = lastLogIndex + 1
		n.volatile.MatchIndex[peer.ID] = 0
	}
}
