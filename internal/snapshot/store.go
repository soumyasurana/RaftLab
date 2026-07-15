package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FileStore manages persisting and loading snapshots to/from a file.
type FileStore struct {
	path string
}

// NewFileStore creates a new FileStore in the given directory.
func NewFileStore(dir string) *FileStore {
	return &FileStore{
		path: filepath.Join(dir, "snapshot.json"),
	}
}

// Save atomically writes the snapshot to disk.
func (s *FileStore) Save(snap Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}

// Load reads the snapshot from disk. Returns false if it does not exist.
func (s *FileStore) Load() (Snapshot, bool, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Snapshot{}, false, nil
		}
		return Snapshot{}, false, err
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return Snapshot{}, false, err
	}

	return snap, true, nil
}
