package wal

import (
	"encoding/binary"
	"os"
	"sync"
)

type Segment struct {
	mu sync.Mutex

	file *os.File

	path string
}

/* Record layout:
| Length(u32) | CRC32(u32)   | Payload      |
*/

func (s *Segment) Write(record []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeLocked(record)
}

func (s *Segment) writeLocked(record []byte) error {
	if len(record) == 0 {
		return ErrEmptyEntry
	}

	length := uint32(len(record))
	crc := checksum(record)

	if err := binary.Write(s.file, binary.BigEndian, length); err != nil {
		return err
	}

	if err := binary.Write(s.file, binary.BigEndian, crc); err != nil {
		return err
	}

	if _, err := s.file.Write(record); err != nil {
		return err
	}

	return nil
}

func (s *Segment) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.file.Sync()
}

func (s *Segment) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.file.Close()
}

// syncLocked flushes the current segment to disk.
// The caller must already hold s.mu.
func (s *Segment) syncLocked() error {
	return s.file.Sync()
}
