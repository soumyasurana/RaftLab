package statemachine

import "encoding/json"

// Encode serializes a state machine command.
func Encode(cmd Command) ([]byte, error) {
	return json.Marshal(cmd)
}

// Decode deserializes a state machine command.
func Decode(data []byte) (Command, error) {
	var cmd Command

	err := json.Unmarshal(data, &cmd)

	return cmd, err
}
