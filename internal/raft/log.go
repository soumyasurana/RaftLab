package raft

import (
	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"github.com/soumyasurana/RaftLab/pkg/types"
)

func protobufEntriesToLogEntries(
	entries []*pb.LogEntry,
) []types.LogEntry {

	result := make([]types.LogEntry, 0, len(entries))

	for _, entry := range entries {

		result = append(result, types.LogEntry{
			Index:   types.LogIndex(entry.Index),
			Term:    types.Term(entry.Term),
			Command: entry.Command,
		})
	}

	return result
}
