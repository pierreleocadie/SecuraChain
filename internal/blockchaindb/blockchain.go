package blockchaindb

import (
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// Blockchain defines an interface for interacting with the blockchain storage.
type Blockchain interface {
	NewBlockchainDB(log *ipfsLog.ZapEventLogger, dbPath string) (*PebbleDB, error)
	SaveBlock(log *ipfsLog.ZapEventLogger, key []byte, b *block.Block) error
	GetBlock(log *ipfsLog.ZapEventLogger, key []byte) (*block.Block, error)
	VerifyIntegrity(log *ipfsLog.ZapEventLogger) bool
	GetLastBlock(log *ipfsLog.ZapEventLogger) *block.Block
	Close(log *ipfsLog.ZapEventLogger) error
}
