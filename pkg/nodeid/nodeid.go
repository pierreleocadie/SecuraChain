// Package nodeid provides functionality to generate and manage unique node identifiers.
package nodeid

import (
	"github.com/google/uuid"
)

// NodeID represents the interface to interact with a node's unique identifier.
type NodeID interface {
	String() string
}

// nodeID is an unexported struct that implements the NodeID interface.
// It encapsulates a UUID to ensure the uniqueness of the identifier.
type nodeID struct {
	identifier uuid.UUID
}

// NewNodeID creates and returns a new NodeID instance with a freshly generated UUID.
func NewNodeID() NodeID {
	return &nodeID{
		identifier: uuid.New(),
	}
}

// String provides a string representation of the NodeID, which is the UUID string.
func (n *nodeID) String() string {
	return n.identifier.String()
}
