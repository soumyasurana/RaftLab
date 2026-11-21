package statemachine

// Apply applies a replicated command.
func (kv *KVStore) Apply(cmd Command) error {

	if err := cmd.Validate(); err != nil {
		return err
	}

	kv.mu.Lock()
	defer kv.mu.Unlock()

	switch cmd.Operation {

	case OpSet:
		kv.data[cmd.Key] = cmd.Value

	case OpDelete:
		delete(kv.data, cmd.Key)
	case OpNoop:
		// Leadership entries advance the log without modifying the store.

	}

	return nil
}
