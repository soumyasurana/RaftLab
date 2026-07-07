package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"raftlab/pkg/types"
)

const (
	DefaultElectionTimeout = 300 * time.Millisecond
	DefaultHeartbeatTimeout = 100 * time.Millisecond
)

type Config struct {
	Node types.NodeConfig
}

// internal structure matching YAML.
type fileConfig struct {
	ID string `yaml:"id"`

	Address string `yaml:"address"`

	DataDir string `yaml:"data_dir"`

	ElectionTimeout string `yaml:"election_timeout"`

	HeartbeatTimeout string `yaml:"heartbeat_timeout"`

	Peers []peerConfig `yaml:"peers"`
}

type peerConfig struct {
	ID string `yaml:"id"`

	Address string `yaml:"address"`
}

// Load loads and validates a configuration file.
func Load(path string) (*Config, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw fileConfig

	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	cfg := &Config{}

	cfg.Node.ID = types.NodeID(raw.ID)
	cfg.Node.Address = raw.Address
	cfg.Node.DataDir = raw.DataDir

	if raw.ElectionTimeout == "" {
		cfg.Node.ElectionTimeout = DefaultElectionTimeout
	} else {

		t, err := time.ParseDuration(raw.ElectionTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid election timeout: %w", err)
		}

		cfg.Node.ElectionTimeout = t
	}

	if raw.HeartbeatTimeout == "" {
		cfg.Node.HeartbeatTimeout = DefaultHeartbeatTimeout
	} else {

		t, err := time.ParseDuration(raw.HeartbeatTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid heartbeat timeout: %w", err)
		}

		cfg.Node.HeartbeatTimeout = t
	}

	for _, peer := range raw.Peers {

		cfg.Node.Peers = append(cfg.Node.Peers, types.Peer{
			ID:      types.NodeID(peer.ID),
			Address: peer.Address,
		})
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks configuration correctness.
func (c *Config) Validate() error {

	if c.Node.ID == "" {
		return errors.New("node id is required")
	}

	if c.Node.Address == "" {
		return errors.New("node address is required")
	}

	if c.Node.DataDir == "" {
		return errors.New("data directory is required")
	}

	if c.Node.ElectionTimeout <= c.Node.HeartbeatTimeout {
		return errors.New("election timeout must be greater than heartbeat timeout")
	}

	seen := make(map[types.NodeID]struct{})

	for _, peer := range c.Node.Peers {

		if peer.ID == "" {
			return errors.New("peer id cannot be empty")
		}

		if peer.Address == "" {
			return fmt.Errorf("peer %s has empty address", peer.ID)
		}

		if peer.ID == c.Node.ID {
			return errors.New("node cannot list itself as a peer")
		}

		if _, exists := seen[peer.ID]; exists {
			return fmt.Errorf("duplicate peer id: %s", peer.ID)
		}

		seen[peer.ID] = struct{}{}
	}

	return nil
}