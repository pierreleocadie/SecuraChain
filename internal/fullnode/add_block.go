package fullnode

import (
	"fmt"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
)

// AddBlockToBlockchain adds a block to the blockchain if it is not already present.
// It returns a boolean indicating whether the block was added, and a message.
func AddBlockToBlockchain(database *pebble.PebbleTransactionDB, blockAnnounced *block.Block) (bool, string) {
	// Use the block's signature as a unique key for storage.
	key := block.ComputeHash(blockAnnounced)

	// Attempt to retrieve the block by its signature to checj for its existence in the database.
	existingBlock, err := database.GetBlock(key)
	if err != nil {
		// An error here means there was a problem accessing the database, not that the block doesn't exist.
		return false, fmt.Sprintf("Error checking for the block existence in the database: %s", err)
	}

	if existingBlock != nil {
		// The bloc already exists in the database, so there's no need to add it again.
		return false, "Block already exists in the blockchain"
	}

	// If the block is not in the blockchain, add it.
	err = database.SaveBlock(key, blockAnnounced)
	if err != nil {
		// An error occurred while trying to save the block to the database.
		return false, fmt.Sprintf("Error adding block to the database: %s", err)
	}

	return true, "Block addded succesfully to the blockchain "
}
