package statemachine

import "sync"

// KVStore is a deterministic replicated state machine.
type KVStore struct {
	mu sync.RWMutex

	data map[string]string
}

// New creates a new KV store.
func New() *KVStore {
	return &KVStore{
		data: make(map[string]string),
	}
}

// Get returns the value associated with a key.
func (kv *KVStore) Get(key string) (string, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	value, ok := kv.data[key]

	return value, ok
}

// Snapshot returns a copy of the current state.
func (kv *KVStore) Snapshot() map[string]string {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	copy := make(map[string]string)

	for k, v := range kv.data {
		copy[k] = v
	}

	return copy
}
