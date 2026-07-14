package metadata

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStore_MissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing_dir")
	file := filepath.Join(dir, "metadata.json")

	store, err := Open(file)
	if err != nil {
		t.Fatalf("expected no error opening in missing dir, got: %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("expected directory to be created, but it was not")
	}

	state, err := store.Load()
	if err != nil {
		t.Fatalf("expected no error loading from missing file, got: %v", err)
	}

	expected := PersistentState{}
	if !reflect.DeepEqual(state, expected) {
		t.Fatalf("expected empty state, got: %+v", state)
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	file := filepath.Join(t.TempDir(), "metadata.json")
	store, err := Open(file)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	state := PersistentState{
		CurrentTerm: 5,
		VotedFor:    "node-1",
	}

	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !reflect.DeepEqual(loaded, state) {
		t.Fatalf("expected state %+v, got %+v", state, loaded)
	}
}

func TestStore_Overwrite(t *testing.T) {
	file := filepath.Join(t.TempDir(), "metadata.json")
	store, err := Open(file)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	state1 := PersistentState{CurrentTerm: 1, VotedFor: "node-1"}
	if err := store.Save(state1); err != nil {
		t.Fatalf("Save 1 failed: %v", err)
	}

	state2 := PersistentState{CurrentTerm: 2, VotedFor: "node-2"}
	if err := store.Save(state2); err != nil {
		t.Fatalf("Save 2 failed: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !reflect.DeepEqual(loaded, state2) {
		t.Fatalf("expected overwritten state %+v, got %+v", state2, loaded)
	}
}

func TestStore_CorruptedJSON(t *testing.T) {
	file := filepath.Join(t.TempDir(), "metadata.json")
	if err := os.WriteFile(file, []byte("{corrupted json"), 0644); err != nil {
		t.Fatalf("failed to write corrupted file: %v", err)
	}

	store, err := Open(file)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatalf("expected error loading corrupted json, got nil")
	}
}

func TestStore_AtomicSave(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "metadata.json")
	store, err := Open(file)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	state := PersistentState{CurrentTerm: 10, VotedFor: "node-3"}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify temp file does not exist after successful save
	tmpFile := file + ".tmp"
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Fatalf("expected tmp file %s to not exist after save", tmpFile)
	}

	// Verify final file exists
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("expected final file %s to exist: %v", file, err)
	}
}
