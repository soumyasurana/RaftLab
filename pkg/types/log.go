package types

// LogEntry represents a single replicated command.
type LogEntry struct {
	Index   LogIndex `json:"index"`
	Term    Term     `json:"term"`
	Command []byte   `json:"command"`
}
