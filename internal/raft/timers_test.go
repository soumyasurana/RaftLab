package raft

import (
	"testing"
	"time"
)

func TestElectionTimer_Randomness(t *testing.T) {
	minTimeout := 150 * time.Millisecond
	maxTimeout := 300 * time.Millisecond

	timer1 := newElectionTimer(minTimeout, maxTimeout)
	timer2 := newElectionTimer(minTimeout, maxTimeout)

	// Sample 10 timeouts from each timer
	const samples = 10
	var sum1, sum2 time.Duration

	for i := 0; i < samples; i++ {
		t1 := timer1.randomTimeout()
		t2 := timer2.randomTimeout()

		if t1 < minTimeout || t1 >= maxTimeout {
			t.Fatalf("timer1 timeout %v out of range [%v, %v)", t1, minTimeout, maxTimeout)
		}
		if t2 < minTimeout || t2 >= maxTimeout {
			t.Fatalf("timer2 timeout %v out of range [%v, %v)", t2, minTimeout, maxTimeout)
		}

		sum1 += t1
		sum2 += t2
	}

	// While it's statistically possible for sums to be equal,
	// it is extremely unlikely with 10 samples and high entropy.
	if sum1 == sum2 {
		t.Fatalf("timer1 and timer2 produced identical sequences of timeouts, missing randomness")
	}
}
