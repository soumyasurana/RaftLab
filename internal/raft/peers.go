package raft

import "github.com/soumyasurana/RaftLab/pkg/types"

// PeerSet tracks active peer topology and network state for a node.
type PeerSet struct {
	peers map[types.NodeID]types.Peer
}

// NewPeerSet initializes a peer set from peer configurations.
func NewPeerSet(peers []types.Peer) *PeerSet {
	set := &PeerSet{
		peers: make(map[types.NodeID]types.Peer, len(peers)),
	}
	for _, peer := range peers {
		set.peers[peer.ID] = peer
	}
	return set
}

// Get returns the peer configuration for the specified node ID.
func (ps *PeerSet) Get(id types.NodeID) (types.Peer, bool) {
	peer, ok := ps.peers[id]
	return peer, ok
}

// List returns all registered peers.
func (ps *PeerSet) List() []types.Peer {
	result := make([]types.Peer, 0, len(ps.peers))
	for _, peer := range ps.peers {
		result = append(result, peer)
	}
	return result
}
