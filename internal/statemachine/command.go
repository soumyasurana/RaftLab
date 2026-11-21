package statemachine

import (
	"encoding/json"
	"fmt"
)

type Operation uint8

const (
	OpSet Operation = iota
	OpDelete
	OpNoop
)

type Command struct {
	Operation Operation `json:"operation"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
}

// Validate checks whether the command is supported by the state machine.
func (c Command) Validate() error {
	switch c.Operation {
	case OpSet, OpDelete, OpNoop:
		return nil
	default:
		return fmt.Errorf("unknown operation %d", c.Operation)
	}
}

// Encode serializes a command.
func (c Command) Encode() ([]byte, error) {
	return json.Marshal(c)
}

// DecodeCommand deserializes a command.
func DecodeCommand(data []byte) (Command, error) {
	var cmd Command

	err := json.Unmarshal(data, &cmd)

	return cmd, err
}
