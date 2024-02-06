package fullnode

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
)

// HandleIncomingBlock handles the logic for processing incoming blocks, including conflict resolution.
func HandleIncomingBlock(incomingBlock *block.Block, blockBuffer map[int64][]*block.Block, database *pebble.PebbleTransactionDB) []byte {
	var timeToWait = 2 * time.Second

	// Add the block to the buffer based on its timestamp.
	timestamp := incomingBlock.Timestamp
	blockBuffer[timestamp] = append(blockBuffer[timestamp], incomingBlock)

	// Artificial delay to allow for more blocks with the same timestamp to arrive.
	time.Sleep(timeToWait)

	blocks := blockBuffer[timestamp]
	var selectedBlock *block.Block

	if len(blocks) > 1 {
		// Randomly choose one block if there are conflicts.
		selectedBlock = blocks[rand.Intn(len(blocks))]
	} else {
		// Proceed normally if there is only one block.
		selectedBlock = blocks[0]
	}

	// Process the selected block.
	if processedSuccessfully := processBlock(selectedBlock, database); processedSuccessfully {
		blockAnnouncedBytes, err := selectedBlock.Serialize()
		if err != nil {
			fmt.Printf("Errior serializing block announcement: %s\n", err)
			return nil
		}
		return blockAnnouncedBytes
	}

	// Clear the buffer for this timestamp after processing.
	delete(blockBuffer, timestamp)
	return nil
}

// processBlock validates and adds a block to the blockchain.
func processBlock(block *block.Block, database *pebble.PebbleTransactionDB) bool {
	if consensus.ValidateBlock(block, nil) {
		// Block is valid, attempt to add it to the blockchain.
		added, message := AddBlockToBlockchain(database, block)
		fmt.Println(message)
		return added
	}
	// Block is invalid
	return false
}
