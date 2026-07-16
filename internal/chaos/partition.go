package chaos

import "github.com/soumyasurana/RaftLab/pkg/types"

func groupForNode(groups [][]types.NodeID, nodeID types.NodeID) (int, bool) {
	for groupIndex, group := range groups {
		for _, member := range group {
			if member == nodeID {
				return groupIndex, true
			}
		}
	}

	return -1, false
}

func partitionAllows(partition Partition, source, target types.NodeID) bool {
	sourceGroup, ok := groupForNode(partition.Groups, source)
	if !ok {
		return false
	}

	targetGroup, ok := groupForNode(partition.Groups, target)
	if !ok {
		return false
	}

	return sourceGroup == targetGroup
}
