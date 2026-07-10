package raft

import (
	"github.com/soumyasurana/RaftLab/pkg/types"
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

// run continuously sends heartbeats while the node remains leader.
func (n *Node) runHeartbeats() {
	ticker := time.NewTicker(n.heartbeat.interval)
	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			n.mu.RLock()

			if n.role != Leader {
				n.mu.RUnlock()
				return
			}

			peers := append(
				[]types.Peer(nil),
				n.config.Node.Peers...,
			)

			n.mu.RUnlock()

			var wg sync.WaitGroup

			for _, peer := range peers {
				peer := peer

				wg.Add(1)

				go func() {
					defer wg.Done()

					n.replicateToPeer(string(peer.ID))
				}()
			}

			wg.Wait()

		case <-n.heartbeat.stopCh:
			return
		}
	}
}
