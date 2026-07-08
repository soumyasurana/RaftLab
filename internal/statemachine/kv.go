package statemachine

import "sync"

// KVStore is an in-memory key-value state machine.
type KVStore struct {
	mu   sync.RWMutex
	data map[string]string
}

// New creates a new key-value store.
func New() *KVStore {
	return &KVStore{
		data: make(map[string]string),
	}
}

// Get returns the value for a key.
func (kv *KVStore) Get(key string) (string, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	value, ok := kv.data[key]
	return value, ok
}

// Delete removes a key.
func (kv *KVStore) Delete(key string) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	delete(kv.data, key)
}
