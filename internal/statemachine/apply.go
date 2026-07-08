package statemachine

import "fmt"

// Apply applies a replicated command.
func (kv *KVStore) Apply(cmd Command) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	switch cmd.Operation {

	case Set:
		kv.data[cmd.Key] = cmd.Value

	case Delete:
		delete(kv.data, cmd.Key)

	default:
		return fmt.Errorf("unknown operation %d", cmd.Operation)
	}

	return nil
}
