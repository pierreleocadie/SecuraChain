package blockchaindb

import (
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// Blockchain represents an interface for interacting with a blockchain.
type Blockchain interface {
	SaveBlock(log *ipfsLog.ZapEventLogger, key []byte, b *block.Block) error
	GetBlock(log *ipfsLog.ZapEventLogger, key []byte) (*block.Block, error)
	VerifyIntegrity(log *ipfsLog.ZapEventLogger) bool
	GetLastBlock(log *ipfsLog.ZapEventLogger) *block.Block
	Close(log *ipfsLog.ZapEventLogger) error
}
