package blockchaindb

import (
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// Blockchain represents an interface for interacting with a blockchain.
type BlockchainDB interface {
	AddBlock(b block.Block) error
	GetBlock(key []byte) (block.Block, error)
	VerifyIntegrity() error
	GetLastBlock() block.Block
	Close() error
}
