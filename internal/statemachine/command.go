package statemachine

type Operation uint8

const (
	Set Operation = iota
	Delete
)

type Command struct {
	Operation Operation
	Key       string
	Value     string
}
