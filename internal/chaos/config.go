package chaos

import (
	"time"

	"github.com/soumyasurana/RaftLab/pkg/types"
)

// Config controls the chaos injector.
type Config struct {
	Enabled               bool
	PacketDropProbability float64
	MinDelay              time.Duration
	MaxDelay              time.Duration
	RandomSeed            int64
	Partitions            []Partition
}

// Partition describes a connectivity split.
//
// Nodes in the same group can communicate. Nodes in different groups cannot.
type Partition struct {
	Groups [][]types.NodeID
}
