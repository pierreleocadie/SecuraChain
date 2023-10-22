// Package idgenerator provides functionality to generate and manage unique identifiers for node and transaction.
package idgen

import (
	"github.com/google/uuid"
)

// Identifier represents the interface to interact with a unique identifier.
type Identifier interface {
	String() string
}

// nodeID is an unexported struct that implements the Identifier interface.
// It encapsulates a UUID to ensure the uniqueness of the identifier.
type identifier struct {
	identifier uuid.UUID
}

// NewIdentifier creates and returns a new identifier instance with a freshly generated UUID.
func NewIdentifier() Identifier {
	return &identifier{
		identifier: uuid.New(),
	}
}

// String provides a string representation of the identifier, which is the UUID string.
func (n *identifier) String() string {
	return n.identifier.String()
}
