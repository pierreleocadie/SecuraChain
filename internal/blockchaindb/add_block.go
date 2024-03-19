package blockchaindb

import (
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// AddBlockToBlockchain adds a block to the blockchain db.
func AddBlockToBlockchain(log *ipfsLog.ZapEventLogger, b *block.Block, db *BlockchainDB) bool {
	// Compute the hash of the block to use as a key in the db
	key := block.ComputeHash(b)

	// Check if the block is the genesis block
	if block.IsGenesisBlock(b) {
		err := db.SaveBlock(log, key, b)
		if err != nil {
			log.Errorf("Failed to add genesis block to the db: %s\n", err)
			return false
		}
		log.Debugln("Genesis block successfully added to the blockchain")
		return true
	}

	// Check if the block already exists in the db
	retrievedBlock, err := db.GetBlock(log, key)
	if err != nil {
		log.Errorf("Failed to check for block existence in the db: %s\n", err)
		return false
	} else if retrievedBlock != nil {
		log.Warnln("Block already exists in the blockchain")
		return false
	}

	if err := db.SaveBlock(log, key, b); err != nil {
		log.Errorf("Failed to add block to the db: %s\n", err)
		return false
	}

	log.Debugln("Block successfully added to the blockchain")
	return true
}
