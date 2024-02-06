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
	// Add the block to the buffer base on his timestamp
	timestamp := incomingBlock.Timestamp
	blockBuffer[timestamp] = append(blockBuffer[timestamp], incomingBlock)

	// Wait a short time to collect more blocks
	time.Sleep(timeToWait) // example of delay

	// Check if there's more than 1 block with the same timestamp
	blocks := blockBuffer[timestamp]
	if len(blocks) > 1 {
		// randomly choose one block to add to the blockchain
		//treat the block
		selectedBlock := blocks[rand.Intn(len(blocks))]
		if processBlock(selectedBlock, database) {
			blockAnnouncedBytes, err := selectedBlock.Serialize()
			if err != nil {
				fmt.Println("Error serializing block announcement : ", err)
			}
			return blockAnnouncedBytes
		}
	} else if len(blocks) == 1 {
		//treat the block normally
		if processBlock(incomingBlock, database) {
			blockAnnouncedBytes, err := incomingBlock.Serialize()
			if err != nil {
				fmt.Println("Error serializing block announcement : ", err)
			}
			return blockAnnouncedBytes
		}
	}
	// Clear the buffer
	delete(blockBuffer, timestamp)
	return nil
}

func processBlock(block *block.Block, database *pebble.PebbleTransactionDB) bool {
	// Process the block
	if consensus.ValidateBlock(block, nil) {
		// Block is valid
		// Add the block to the blockchain
		added, message := AddBlockToBlockchain(database, block)
		if added {
			fmt.Println(message)
			return true
		} else {
			fmt.Println(message)
			return false
		}
	} else {
		// Block is invalid
		return false
	}
}
