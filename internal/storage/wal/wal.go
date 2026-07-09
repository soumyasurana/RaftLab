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

// ReadAll loads every entry from disk.
func (w *WAL) ReadAll() ([]types.LogEntry, error) {
	w.segment.mu.Lock()
	defer w.segment.mu.Unlock()

	if _, err := w.segment.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var entries []types.LogEntry

	for {
		var length uint32
		var crc uint32

		if err := binary.Read(w.segment.file, binary.BigEndian, &length); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if err := binary.Read(w.segment.file, binary.BigEndian, &crc); err != nil {
			return nil, err
		}

		payload := make([]byte, length)

		if _, err := io.ReadFull(w.segment.file, payload); err != nil {
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

func (w *WAL) Sync() error {
	return w.segment.Sync()
}

func (w *WAL) Close() error {
	return w.segment.Close()
}
