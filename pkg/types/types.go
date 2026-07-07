package types

import "time"

// NodeID uniquely identifies a Raft node.
type NodeID string

// Term represents a Raft election term.
type Term uint64

// LogIndex represents an index into the replicated log.
type LogIndex uint64

// NodeState represents the current role of a Raft node.
type NodeState int

const (
	Follower NodeState = iota
	Candidate
	Leader
)

func (s NodeState) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

// Peer represents another node in the cluster.
type Peer struct {
	ID      NodeID `yaml:"id"`
	Address string `yaml:"address"`
}

// NodeConfig contains the configuration for a node.
type NodeConfig struct {
	ID               NodeID        `yaml:"id"`
	Address          string        `yaml:"address"`
	DataDir          string        `yaml:"data_dir"`
	ElectionTimeout  time.Duration `yaml:"election_timeout"`
	HeartbeatTimeout time.Duration `yaml:"heartbeat_timeout"`
	Peers            []Peer        `yaml:"peers"`
}