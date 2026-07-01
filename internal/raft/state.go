package raft

type State int

const (
	Follower State = iota
	Candidate
	Leader
)

// TODO: implement state transitions
