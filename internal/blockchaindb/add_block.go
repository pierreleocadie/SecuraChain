package blockchaindb

import (
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// AddBlockToBlockchain adds a block to the blockchain database.
func AddBlockToBlockchain(log *ipfsLog.ZapEventLogger, b *block.Block, database *BlockchainDB) bool {
	// Compute the hash of the block to use as a key in the database
	key := block.ComputeHash(b)

	// Check if the block is the genesis block
	if block.IsGenesisBlock(b) {
		err := database.SaveBlock(key, b)
		if err != nil {
			log.Errorf("Failed to add genesis block to the database: %s\n", err)
			return false
		}
		log.Debugln("Genesis block successfully added to the blockchain")
		return true
	}

	// Check if the block already exists in the database
	retrievedBlock, err := database.GetBlock(key)
	if err != nil {
		log.Errorf("Failed to check for block existence in the database: %s\n", err)
		return false
	} else if retrievedBlock != nil {
		log.Debugln("Block already exists in the blockchain")
		return false
	}

	if err := database.SaveBlock(key, b); err != nil {
		log.Errorf("Failed to add block to the database: %s\n", err)
		return false
	}

	log.Debugln("Block successfully added to the blockchain")
	return true
}
