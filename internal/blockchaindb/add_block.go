package blockchaindb

import (
	"fmt"

	"github.com/cockroachdb/pebble"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// addGenesisBlock adds the genesis block to the blockchain database.
func addGenesisBlock(log *ipfsLog.ZapEventLogger, b *block.Block, db *PebbleDB) error {
	// Compute the hash of the block to use as a key in the db
	key := block.ComputeHash(b)

	err := db.SaveBlock(log, key, b)
	if err != nil {
		log.Errorf("Failed to add genesis block to the db: %s\n", err)
		return fmt.Errorf("failed to add genesis block to the db: %s", err)
	}
	log.Debugln("Genesis block successfully added to the blockchain")
	return nil
}

// addNormalBlock adds a normal block to the blockchain database.
func addNormalBlock(log *ipfsLog.ZapEventLogger, b *block.Block, db *PebbleDB) error {
	key := block.ComputeHash(b)

	// Check if the block already exists in the db
	_, err := db.GetBlock(log, key)
	if err != nil {
		if err == pebble.ErrNotFound {
			log.Errorf("Block not found in the db: %s\n", err)

			if err := db.SaveBlock(log, key, b); err != nil {
				log.Errorf("Failed to add block to the db: %s\n", err)
				return fmt.Errorf("failed to add block to the db: %s", err)
			}

			log.Debugln("Block successfully added to the blockchain")
			return nil
		}

		log.Errorf("Failed to retrieve block from the db: %s\n", err)
		return fmt.Errorf("failed to retrieve block from the db: %s", err)

	}
	log.Warnln("Block already exists in the blockchain")
	return fmt.Errorf("block already exists in the blockchain")
}

// AddBlockToBlockchain adds a block to the blockchain database.
func AddBlockToBlockchain(log *ipfsLog.ZapEventLogger, b *block.Block, db *PebbleDB) error {
	if block.IsGenesisBlock(b) {
		return addGenesisBlock(log, b, db)
	}
	return addNormalBlock(log, b, db)
}
