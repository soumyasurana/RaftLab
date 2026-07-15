package snapshot

import (
	"reflect"
	"testing"
)

func TestFileStore(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	// Loading when non-existent
	snap, exists, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected error loading non-existent snapshot: %v", err)
	}
	if exists {
		t.Fatal("expected exists to be false")
	}

	// Saving a snapshot
	expectedSnap := Snapshot{
		LastIncludedIndex: 10,
		LastIncludedTerm:  2,
		Data:              []byte("test data"),
	}

	if err := store.Save(expectedSnap); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	// Loading existing snapshot
	snap, exists, err = store.Load()
	if err != nil {
		t.Fatalf("unexpected error loading snapshot: %v", err)
	}
	if !exists {
		t.Fatal("expected exists to be true")
	}

	if !reflect.DeepEqual(snap, expectedSnap) {
		t.Fatalf("loaded snapshot %+v does not match expected %+v", snap, expectedSnap)
	}

	// Replacing snapshot
	expectedSnap.LastIncludedIndex = 20
	expectedSnap.LastIncludedTerm = 3
	expectedSnap.Data = []byte("new data")

	if err := store.Save(expectedSnap); err != nil {
		t.Fatalf("failed to replace snapshot: %v", err)
	}

	snap, exists, err = store.Load()
	if err != nil {
		t.Fatalf("unexpected error loading replaced snapshot: %v", err)
	}
	if !exists {
		t.Fatal("expected exists to be true")
	}

	if !reflect.DeepEqual(snap, expectedSnap) {
		t.Fatalf("loaded replaced snapshot %+v does not match expected %+v", snap, expectedSnap)
	}
}
