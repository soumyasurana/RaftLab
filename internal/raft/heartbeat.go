package raft

import (
	"context"
	"sync"
	"time"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/pkg/types"
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
	ticker := time.NewTicker(
		n.heartbeat.interval,
	)
	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			n.mu.RLock()

			if n.role != Leader {
				n.mu.RUnlock()
				return
			}

			term := n.persistent.CurrentTerm

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

					ctx, cancel := context.WithTimeout(
						context.Background(),
						200*time.Millisecond,
					)
					defer cancel()

					response, err := n.rpcClient.AppendEntries(
						ctx,
						string(peer.ID),
						&pb.AppendEntriesRequest{
							Term:         term,
							LeaderId:     string(n.config.Node.ID),
							LeaderCommit: n.volatile.CommitIndex,
						},
					)

					if err != nil {
						return
					}

					if response.Term > term {

						n.mu.Lock()

						if response.Term > n.persistent.CurrentTerm {
							n.becomeFollowerLocked(response.Term)
						}

						n.mu.Unlock()
					}
				}()
			}

			wg.Wait()

		case <-n.heartbeat.stopCh:
			return
		}
	}
}
