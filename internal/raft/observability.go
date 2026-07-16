package raft

// Role returns the node's current role.
func (n *Node) Role() Role {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.role
}

// Term returns the node's current term.
func (n *Node) Term() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.persistent.CurrentTerm
}

// CommitIndex returns the committed log index.
func (n *Node) CommitIndex() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.volatile.CommitIndex
}

// LastApplied returns the last applied log index.
func (n *Node) LastApplied() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.volatile.LastApplied
}

// StateMachineValue returns a key from the replicated state machine.
func (n *Node) StateMachineValue(key string) (string, bool) {
	return n.stateMachine.Get(key)
}
