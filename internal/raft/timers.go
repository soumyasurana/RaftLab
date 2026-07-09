package raft

import (
	"math/rand"
	"sync"
	"time"
)

// electionTimer manages randomized Raft election timeouts.
type electionTimer struct {
	minTimeout time.Duration
	maxTimeout time.Duration

	resetCh chan struct{}
	stopCh  chan struct{}

	stopOnce sync.Once
}

// newElectionTimer creates an election timer with randomized timeouts.
func newElectionTimer(
	minTimeout time.Duration,
	maxTimeout time.Duration,
) *electionTimer {
	return &electionTimer{
		minTimeout: minTimeout,
		maxTimeout: maxTimeout,
		resetCh:    make(chan struct{}, 1),
		stopCh:     make(chan struct{}),
	}
}

// run starts the election timer.
// The callback is invoked whenever the randomized election timeout expires.
func (t *electionTimer) run(onTimeout func()) {
	timer := time.NewTimer(t.randomTimeout())
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			onTimeout()
			resetTimer(timer, t.randomTimeout())

		case <-t.resetCh:
			resetTimer(timer, t.randomTimeout())

		case <-t.stopCh:
			return
		}
	}
}

// reset requests a new randomized election timeout.
func (t *electionTimer) reset() {
	select {
	case t.resetCh <- struct{}{}:
	default:
	}
}

// stop terminates the election timer.
func (t *electionTimer) stop() {
	t.stopOnce.Do(func() {
		close(t.stopCh)
	})
}

// randomTimeout returns a duration in [minTimeout, maxTimeout).
func (t *electionTimer) randomTimeout() time.Duration {
	timeoutRange := t.maxTimeout - t.minTimeout

	if timeoutRange <= 0 {
		return t.minTimeout
	}

	return t.minTimeout +
		time.Duration(rand.Int63n(int64(timeoutRange)))
}

// resetTimer safely stops, drains, and resets a timer.
func resetTimer(timer *time.Timer, timeout time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}

	timer.Reset(timeout)
}
