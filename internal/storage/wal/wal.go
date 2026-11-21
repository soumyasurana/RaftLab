package wal

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"

	"github.com/soumyasurana/RaftLab/pkg/types"
)

type WAL struct {
	segment *Segment
}

// Open creates or opens a WAL file.
func Open(path string) (*WAL, error) {
	if err := os.MkdirAll(
		filepath.Dir(path),
		0755,
	); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, err
	}

	segment := &Segment{
		file: file,
		path: path,
	}

	return &WAL{
		segment: segment,
	}, nil
}

// Append persists a log entry.
func (w *WAL) Append(entry types.LogEntry) error {

	record := FromLogEntry(entry)

	data, err := record.MarshalBinary()
	if err != nil {
		return err
	}

	if err := w.segment.Write(data); err != nil {
		return err
	}

	return w.segment.Sync()
}

// readAllLocked loads every entry from disk.
// The caller must hold w.segment.mu.
func (w *WAL) readAllLocked() ([]types.LogEntry, error) {

	if _, err := w.segment.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	info, err := w.segment.file.Stat()
	if err != nil {
		return nil, err
	}
	remaining := info.Size()
	var entries []types.LogEntry

	for {

		var length uint32
		var crc uint32

		if err := binary.Read(
			w.segment.file,
			binary.BigEndian,
			&length,
		); err != nil {

			if err == io.EOF {
				break
			}

			return nil, err
		}

		if err := binary.Read(
			w.segment.file,
			binary.BigEndian,
			&crc,
		); err != nil {
			return nil, err
		}

		remaining -= 8 + int64(length)
		if remaining < 0 {
			return nil, io.ErrUnexpectedEOF
		}
		payload := make([]byte, length)

		if _, err := io.ReadFull(
			w.segment.file,
			payload,
		); err != nil {
			return nil, err
		}

		if !verifyChecksum(payload, crc) {
			return nil, ErrCorruptedEntry
		}

		var record Entry

		if err := record.UnmarshalBinary(payload); err != nil {
			return nil, err
		}

		entries = append(entries, record.ToLogEntry())
	}

	return entries, nil
}

// ReadAll loads every entry from disk.
func (w *WAL) ReadAll() ([]types.LogEntry, error) {
	w.segment.mu.Lock()
	defer w.segment.mu.Unlock()

	return w.readAllLocked()
}

func (w *WAL) Sync() error {
	return w.segment.Sync()
}

func (w *WAL) Close() error {
	return w.segment.Close()
}

// LastEntry returns the latest log entry.
func (w *WAL) LastEntry() (types.LogEntry, bool, error) {
	entries, err := w.ReadAll()
	if err != nil {
		return types.LogEntry{}, false, err
	}

	if len(entries) == 0 {
		return types.LogEntry{}, false, nil
	}

	return entries[len(entries)-1], true, nil
}

// EntryAt returns the entry at the given Raft index.
func (w *WAL) EntryAt(index uint64) (types.LogEntry, bool, error) {
	entries, err := w.ReadAll()
	if err != nil {
		return types.LogEntry{}, false, err
	}

	if index == 0 || len(entries) == 0 {
		return types.LogEntry{}, false, nil
	}

	firstIndex := uint64(entries[0].Index)
	lastIndex := uint64(entries[len(entries)-1].Index)

	if index < firstIndex || index > lastIndex {
		return types.LogEntry{}, false, nil
	}

	offset := index - firstIndex
	if offset < uint64(len(entries)) && uint64(entries[offset].Index) == index {
		return entries[offset], true, nil
	}

	for _, entry := range entries {
		if uint64(entry.Index) == index {
			return entry, true, nil
		}
	}

	return types.LogEntry{}, false, nil
}

// EntriesFrom returns every entry starting from index.
func (w *WAL) EntriesFrom(index uint64) ([]types.LogEntry, error) {
	entries, err := w.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return []types.LogEntry{}, nil
	}

	if index == 0 {
		return entries, nil
	}

	firstIndex := uint64(entries[0].Index)
	lastIndex := uint64(entries[len(entries)-1].Index)

	if index < firstIndex {
		return entries, nil
	}

	if index > lastIndex {
		return []types.LogEntry{}, nil
	}

	offset := index - firstIndex
	if offset < uint64(len(entries)) && uint64(entries[offset].Index) == index {
		return entries[offset:], nil
	}

	for i, entry := range entries {
		if uint64(entry.Index) >= index {
			return entries[i:], nil
		}
	}

	return []types.LogEntry{}, nil
}

// TruncateAfter removes all entries after the given index.
func (w *WAL) TruncateAfter(index uint64) error {

	w.segment.mu.Lock()
	defer w.segment.mu.Unlock()

	entries, err := w.readAllLocked()
	if err != nil {
		return err
	}

	var retained []types.LogEntry

	for _, entry := range entries {

		if uint64(entry.Index) <= index {
			retained = append(retained, entry)
		}
	}

	if len(retained) == len(entries) {
		return nil
	}
	return w.replaceEntriesLocked(retained)
}

// AppendEntries appends multiple log entries.
func (w *WAL) AppendEntries(entries []types.LogEntry) error {
	for _, entry := range entries {
		if err := w.Append(entry); err != nil {
			return err
		}
	}

	return nil
}

// TruncateBefore removes all entries up to and including the given index.
// Used during log compaction.
func (w *WAL) TruncateBefore(index uint64) error {

	w.segment.mu.Lock()
	defer w.segment.mu.Unlock()

	entries, err := w.readAllLocked()
	if err != nil {
		return err
	}

	var retained []types.LogEntry

	for _, entry := range entries {
		if uint64(entry.Index) > index {
			retained = append(retained, entry)
		}
	}

	if len(retained) == len(entries) {
		return nil
	}
	return w.replaceEntriesLocked(retained)
}

// replaceEntriesLocked builds and syncs a replacement before changing the live WAL.
// The caller must hold w.segment.mu.
func (w *WAL) replaceEntriesLocked(entries []types.LogEntry) error {
	file, err := os.CreateTemp(filepath.Dir(w.segment.path), ".raft-wal-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()

	replacement := Segment{file: file}
	for _, entry := range entries {
		record := FromLogEntry(entry)
		data, err := record.MarshalBinary()
		if err != nil {
			return err
		}
		if err := replacement.writeLocked(data); err != nil {
			return err
		}
	}
	if err := file.Sync(); err != nil {
		return err
	}
	// Open the replacement in append mode before swapping it into place.
	appendFile, err := os.OpenFile(file.Name(), os.O_RDWR|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	if err := os.Rename(file.Name(), w.segment.path); err != nil {
		_ = appendFile.Close()
		return err
	}

	previous := w.segment.file
	w.segment.file = appendFile
	_ = previous.Close()
	return nil
}
