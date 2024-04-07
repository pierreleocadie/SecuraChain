package visualisation

import "slices"

type DiffData struct {
	Added   []Data `json:"added"`   // Nodes that were added
	Removed []Data `json:"removed"` // Nodes that were removed
	Updated []Data `json:"updated"` // Nodes that were updated
}

func NewDiffData(oldData, newData []Data) DiffData {
	return DiffData{
		Added:   FindAddedNodes(oldData, newData),
		Removed: FindRemovedNodes(oldData, newData),
		Updated: FindUpdatedNodes(oldData, newData),
	}
}

func FindAddedNodes(oldData, newData []Data) []Data {
	added := make([]Data, 0)
	oldDataMap := make(map[string]bool)

	for _, d := range oldData {
		oldDataMap[d.PeerID] = true
	}

	for _, d := range newData {
		if _, exists := oldDataMap[d.PeerID]; !exists {
			added = append(added, d)
		}
	}

	return added
}

func FindRemovedNodes(oldData, newData []Data) []Data {
	removed := make([]Data, 0)
	newDataMap := make(map[string]bool)

	for _, d := range newData {
		newDataMap[d.PeerID] = true
	}

	for _, d := range oldData {
		if _, exists := newDataMap[d.PeerID]; !exists {
			removed = append(removed, d)
		}
	}

	return removed
}

func FindUpdatedNodes(oldData, newData []Data) []Data {
	var updated []Data

	oldDataMap := make(map[string]Data)
	for _, node := range oldData {
		oldDataMap[node.PeerID] = node
	}

	for _, node := range newData {
		if oldNode, ok := oldDataMap[node.PeerID]; ok && slices.Equal(oldNode.ConnectedPeers, node.ConnectedPeers) {
			updated = append(updated, node)
		}
	}

	return updated
}
