// Package nodeid provides functionality to generate and manage unique node identifiers.
package nodeid

import (
	"github.com/google/uuid"
)

// NodeID represents a unique identifier for a node.
// It encapsulates a UUID to ensure the uniqueness of the identifier.
type NodeID struct {
	identifier uuid.UUID
}

// NewNodeID creates and returns a new NodeID instance with a freshly generated UUID.
func NewNodeID() *NodeID {
	return &NodeID{
		identifier: uuid.New(),
	}
}

// String provides a string representation of the NodeID, which is the UUID string.
func (nodeId NodeID) String() string {
	return nodeId.identifier.String()
}
