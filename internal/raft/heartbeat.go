package raft

import (
	"sync"
	"time"
)

type heartbeatManager struct {
	interval time.Duration

	stopCh chan struct{}

	stopOnce sync.Once
}

func newHeartbeatManager(interval time.Duration) *heartbeatManager {
	return &heartbeatManager{
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (h *heartbeatManager) stop() {
	h.stopOnce.Do(func() {
		close(h.stopCh)
	})
}

// runHeartbeats sends replication requests for the captured leadership term.
func (n *Node) runHeartbeats(manager *heartbeatManager, term uint64) {
	ticker := time.NewTicker(manager.interval)
	defer ticker.Stop()

	for {
		n.replicateToPeers(term)
		select {
		case <-ticker.C:
		case <-manager.stopCh:
			return
		}
	}
}
