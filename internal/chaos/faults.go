package chaos

import "errors"

var (
	ErrDropped      = errors.New("chaos: rpc dropped")
	ErrPartitioned  = errors.New("chaos: rpc blocked by partition")
	ErrDisconnected = errors.New("chaos: node disconnected")
	ErrCrashed      = errors.New("chaos: node crashed")
)

type rpcMethod string

const (
	methodRequestVote     rpcMethod = "RequestVote"
	methodAppendEntries   rpcMethod = "AppendEntries"
	methodInstallSnapshot rpcMethod = "InstallSnapshot"
)
