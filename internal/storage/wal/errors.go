package wal

import "errors"

var (
	ErrCorruptedEntry = errors.New("corrupted wal entry")
	ErrSegmentClosed  = errors.New("wal segment is closed")
	ErrEmptyEntry     = errors.New("empty wal entry")
)
