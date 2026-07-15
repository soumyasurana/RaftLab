package snapshot

// Snapshot represents a compacted state machine state.
// It includes metadata (LastIncludedIndex, LastIncludedTerm) and the state machine data.
type Snapshot struct {
	LastIncludedIndex uint64 `json:"last_included_index"`
	LastIncludedTerm  uint64 `json:"last_included_term"`
	Data              []byte `json:"data"`
}
