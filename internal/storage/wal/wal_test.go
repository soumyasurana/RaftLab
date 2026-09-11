package wal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soumyasurana/RaftLab/pkg/types"
)

func TestAppendAndReadSingleEntry(t *testing.T) {
	dir := t.TempDir()

	wal, err := Open(filepath.Join(dir, "wal.log"))
	if err != nil {
		t.Fatal(err)
	}

	defer wal.Close()

	entry := types.LogEntry{
		Index:   1,
		Term:    1,
		Command: []byte("SET x 10"),
	}

	if err := wal.Append(entry); err != nil {
		t.Fatal(err)
	}

	entries, err := wal.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Index != entry.Index {
		t.Fatal("index mismatch")
	}

	if entries[0].Term != entry.Term {
		t.Fatal("term mismatch")
	}

	if string(entries[0].Command) != string(entry.Command) {
		t.Fatal("command mismatch")
	}
}

func TestAppendMultipleEntries(t *testing.T) {
	dir := t.TempDir()

	wal, err := Open(filepath.Join(dir, "wal.log"))
	if err != nil {
		t.Fatal(err)
	}

	defer wal.Close()

	for i := uint64(1); i <= 100; i++ {
		err := wal.Append(types.LogEntry{
			Index:   types.LogIndex(i),
			Term:    1,
			Command: []byte("command"),
		})

		if err != nil {
			t.Fatal(err)
		}
	}

	entries, err := wal.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 100 {
		t.Fatalf("expected 100 entries got %d", len(entries))
	}
}

func TestRecoveryAfterReopen(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "wal.log")

	{
		wal, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}

		err = wal.Append(types.LogEntry{
			Index:   1,
			Term:    5,
			Command: []byte("hello"),
		})

		if err != nil {
			t.Fatal(err)
		}

		wal.Close()
	}

	{
		wal, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}

		defer wal.Close()

		entries, err := wal.ReadAll()
		if err != nil {
			t.Fatal(err)
		}

		if len(entries) != 1 {
			t.Fatal("recovery failed")
		}

		if entries[0].Term != 5 {
			t.Fatal("wrong term after recovery")
		}
	}
}

func TestDetectCorruption(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "wal.log")

	wal, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	err = wal.Append(types.LogEntry{
		Index:   1,
		Term:    1,
		Command: []byte("abc"),
	})

	if err != nil {
		t.Fatal(err)
	}

	wal.Close()

	file, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = file.WriteAt([]byte{0xff, 0xff, 0xff}, 20)
	if err != nil {
		t.Fatal(err)
	}

	file.Close()

	wal, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer wal.Close()

	_, err = wal.ReadAll()

	if err == nil {
		t.Fatal("expected corruption error")
	}
}

func TestTruncateBeforeAndReadIndexed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal.log")

	wal, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()

	for i := uint64(1); i <= 10; i++ {
		if err := wal.Append(types.LogEntry{
			Index:   types.LogIndex(i),
			Term:    1,
			Command: []byte("cmd"),
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := wal.TruncateBefore(5); err != nil {
		t.Fatal(err)
	}

	// Index 5 and below should not be present
	if _, ok, _ := wal.EntryAt(5); ok {
		t.Fatalf("expected entry 5 to be truncated")
	}

	// Index 6 should be present
	entry, ok, err := wal.EntryAt(6)
	if err != nil || !ok || uint64(entry.Index) != 6 {
		t.Fatalf("expected entry 6, got ok=%v entry=%+v err=%v", ok, entry, err)
	}

	// EntriesFrom(6) should return 5 entries (6..10)
	entries, err := wal.EntriesFrom(6)
	if err != nil || len(entries) != 5 {
		t.Fatalf("expected 5 entries from index 6, got %d (err: %v)", len(entries), err)
	}
}
