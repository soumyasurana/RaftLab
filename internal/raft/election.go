package raft

import (
	"context"
	"log"
	"sync"
	"time"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/internal/statemachine"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

const defaultVoteRPCTimeout = 200 * time.Millisecond

// handleElectionTimeout starts a new election.
func (n *Node) handleElectionTimeout() {
	n.mu.Lock()

	if n.role == Leader || n.stopped {
		n.mu.Unlock()
		return
	}

	previousRole, previousState := n.role, n.persistent
	n.role = Candidate

	n.persistent.CurrentTerm++
	n.persistent.VotedFor = n.config.Node.ID

	if err := n.persistLocked(); err != nil {
		n.role, n.persistent = previousRole, previousState
		n.mu.Unlock()
		log.Printf(
			"persist metadata: %v",
			err,
		)
		return
	}

	electionTerm := n.persistent.CurrentTerm
	log.Printf(
		"node=%s started election term=%d",
		n.config.Node.ID,
		electionTerm,
	)

	lastLogIndex, lastLogTerm, err := n.lastLogInfoLocked()

	if err != nil {
		n.mu.Unlock()
		return
	}

	peers := append(
		[]types.Peer(nil),
		n.config.Node.Peers...,
	)

	n.mu.Unlock()

	// A candidate always votes for itself.
	votes := 1

	clusterSize := len(peers) + 1
	quorum := clusterSize/2 + 1

	// A one-node cluster immediately elects itself.
	if votes >= quorum {
		n.becomeLeader(electionTerm)
		return
	}

	var wg sync.WaitGroup

	var votesMu sync.Mutex

	for _, peer := range peers {
		peer := peer

		wg.Add(1)

		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(
				context.Background(),
				defaultVoteRPCTimeout,
			)
			defer cancel()

			response, err := n.rpcClient.RequestVote(
				ctx,
				string(peer.ID),
				&pb.RequestVoteRequest{
					Term:         electionTerm,
					CandidateId:  string(n.config.Node.ID),
					LastLogIndex: lastLogIndex,
					LastLogTerm:  lastLogTerm,
				},
			)
			if err != nil {
				log.Printf(
					"node=%s vote request failed peer=%s term=%d error=%v",
					n.config.Node.ID,
					peer.ID,
					electionTerm,
					err,
				)

				return
			}

			n.mu.Lock()
			if n.stopped {
				n.mu.Unlock()
				return
			}

			// A response from a newer term invalidates this election.
			if response.Term > n.persistent.CurrentTerm {
				if err := n.becomeFollowerLocked(response.Term); err != nil {
					log.Printf("persist newer term: %v", err)
				} else {
					n.electionTimer.reset()
				}
				n.mu.Unlock()
				return
			}

			// Ignore responses from an election that is no longer active.
			if n.role != Candidate ||
				n.persistent.CurrentTerm != electionTerm {
				n.mu.Unlock()
				return
			}

			n.mu.Unlock()

			if !response.VoteGranted {
				return
			}
			log.Printf(
				"node=%s received vote peer=%s term=%d",
				n.config.Node.ID,
				peer.ID,
				electionTerm,
			)

			votesMu.Lock()
			votes++
			wonElection := votes == quorum
			votesMu.Unlock()

			if wonElection {
				n.becomeLeader(electionTerm)
			}
		}()
	}

	wg.Wait()

	n.mu.Lock()
	if n.role == Candidate && n.persistent.CurrentTerm == electionTerm {
		n.metrics.ElectionsLost++
	}
	n.mu.Unlock()
}

// becomeLeader promotes the node only if the election is still current.
func (n *Node) becomeLeader(electionTerm uint64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.role != Candidate || n.stopped {
		return
	}

	if n.persistent.CurrentTerm != electionTerm {
		return
	}

	lastIndex, lastTerm, err := n.lastLogInfoLocked()
	if err != nil {
		return
	}

	// A current-term entry lets a new leader commit an inherited log even
	// when clients issue no new writes after a failover or restart.
	if lastIndex > n.volatile.CommitIndex && lastTerm < electionTerm {
		command, err := (statemachine.Command{Operation: statemachine.OpNoop}).Encode()
		if err != nil {
			return
		}
		if err := n.wal.Append(types.LogEntry{
			Index:   types.LogIndex(lastIndex + 1),
			Term:    types.Term(electionTerm),
			Command: command,
		}); err != nil {
			log.Printf("append leadership no-op: %v", err)
			return
		}
	}

	n.role = Leader
	n.leaderID = n.config.Node.ID
	n.metrics.ElectionsWon++
	n.metrics.LeaderChanges++
	n.initializeLeaderState(lastIndex)
	n.advanceCommitIndex()

	log.Printf(
		"node=%s became leader term=%d",
		n.config.Node.ID,
		electionTerm,
	)
	n.heartbeat = newHeartbeatManager(
		n.config.Node.HeartbeatTimeout,
	)

	go n.runHeartbeats(n.heartbeat, electionTerm)
}
