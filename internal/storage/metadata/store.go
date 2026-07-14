package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/soumyasurana/RaftLab/pkg/types"
)

type PersistentState struct {
	CurrentTerm types.Term   `json:"current_term"`
	VotedFor    types.NodeID `json:"voted_for"`
}

type Store struct {
	path string
}

func Open(path string) (*Store, error) {

	if err := os.MkdirAll(
		filepath.Dir(path),
		0755,
	); err != nil {
		return nil, err
	}

	return &Store{
		path: path,
	}, nil
}

// Load loads persistent Raft metadata.
// Missing files are treated as an empty state.
func (s *Store) Load() (PersistentState, error) {

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return PersistentState{}, nil
	}

	if err != nil {
		return PersistentState{}, err
	}

	var state PersistentState

	if err := json.Unmarshal(data, &state); err != nil {
		return PersistentState{}, err
	}

	return state, nil
}

// Save atomically persists Raft metadata.
func (s *Store) Save(state PersistentState) error {

	data, err := json.MarshalIndent(
		state,
		"",
		"  ",
	)
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"

	if err := os.WriteFile(
		tmp,
		data,
		0644,
	); err != nil {
		return err
	}

	return os.Rename(tmp, s.path)
}
