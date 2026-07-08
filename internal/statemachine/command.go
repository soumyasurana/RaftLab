package statemachine

import "encoding/json"

type Operation uint8

const (
	OpSet Operation = iota
	OpDelete
)

type Command struct {
	Operation Operation `json:"operation"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
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
