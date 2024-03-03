package pebble

import (
	"bytes"
	"fmt"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// AddBlockToBlockchain adds a block to the blockchain if it is not already present.
// It returns a boolean indicating whether the block was added, and a message.
func AddBlockToBlockchain(b *block.Block, database *PebbleDB) (bool, string) {
	// Use the block's signature as a unique key for storage.
	key := block.ComputeHash(b)

	// Serialize the original block
	originalBlockBytes, err := b.Serialize()
	if err != nil {
		return false, fmt.Sprintf("Error serializing the original block: %v", err)
	}

	// Attempt to retrieve the block by its signature to check for its existence in the database.
	retrievedBlock, err := database.GetBlock(key)
	if err != nil {
		return false, fmt.Sprintf("Error checking for the block existence in the database or Block already exists in the blockchain: %s", err)
	}

	// Serialize the retrieved block
	retrievedBlockBytes, err := retrievedBlock.Serialize()
	if err != nil {
		return false, "Error serializing the retrieved block: " + err.Error()
	}

	// Compare the original block with the retrieved block
	if bytes.Equal(originalBlockBytes, retrievedBlockBytes) {
		return false, "Block already exists in the blockchain"
	}

	// If the block is not in the blockchain, add it.
	err = database.SaveBlock(key, b)
	if err != nil {
		// An error occurred while trying to save the block to the database.
		return false, "Error adding block to the database: " + err.Error()
	}

	return true, "Block addded succesfully to the blockchain"
}
