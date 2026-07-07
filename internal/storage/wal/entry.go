package wal

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/soumyasurana/RaftLab/pkg/types"
)

// Entry represents a single WAL record.
type Entry struct {
	Index   uint64
	Term    uint64
	Command []byte
}

// FromLogEntry converts a shared log entry into a WAL entry.
func FromLogEntry(entry types.LogEntry) Entry {
	return Entry{
		Index:   uint64(entry.Index),
		Term:    uint64(entry.Term),
		Command: entry.Command,
	}
}

// ToLogEntry converts a WAL entry back to the shared type.
func (e Entry) ToLogEntry() types.LogEntry {
	return types.LogEntry{
		Index:   types.LogIndex(e.Index),
		Term:    types.Term(e.Term),
		Command: e.Command,
	}
}

/* MarshalBinary serializes an entry.
 Format:
 | Index   uint64 |
 | Term    uint64 |
 | CmdLen  uint32 |
 | Command bytes  |
*/
func (e Entry) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	fields := []any{
		e.Index,
		e.Term,
		uint32(len(e.Command)),
	}

	for _, field := range fields {
		if err := binary.Write(&buf, binary.BigEndian, field); err != nil {
			return nil, err
		}
	}

	if _, err := buf.Write(e.Command); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary deserializes an entry.
func (e *Entry) UnmarshalBinary(data []byte) error {
	reader := bytes.NewReader(data)

	if err := binary.Read(reader, binary.BigEndian, &e.Index); err != nil {
		return err
	}

	if err := binary.Read(reader, binary.BigEndian, &e.Term); err != nil {
		return err
	}

	var commandLength uint32

	if err := binary.Read(reader, binary.BigEndian, &commandLength); err != nil {
		return err
	}

	e.Command = make([]byte, commandLength)

	n, err := reader.Read(e.Command)
	if err != nil {
		return err
	}

	if uint32(n) != commandLength {
		return fmt.Errorf("expected %d command bytes, read %d", commandLength, n)
	}

	return nil
}
