package fullnode

import (
	"fmt"
	"sort"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
)

// HandleIncomingBlock handles the logic for processing incoming blocks, including conflict resolution.
func HandleIncomingBlock(incomingBlock *block.Block, blockBuffer map[int64][]*block.Block, database *pebble.PebbleDB) ([]*block.Block, error) {
	var timeToWait = 2 * time.Second

	// verify that the blocks got's the same hash before adding it to the buffer

	// Add the block to the buffer based on its timestamp.
	timestamp := incomingBlock.Timestamp
	blockBuffer[timestamp] = append(blockBuffer[timestamp], incomingBlock)

	// Artificial delay to allow for more blocks with the same timestamp to arrive.
	time.Sleep(timeToWait)

	blocks := blockBuffer[timestamp]

	if len(blocks) > 1 {
		// Return all blocks with the same timestamp for the minor node to select based on the longest chain
		return blocks, nil
	}

	// Proceed normally if there is only one block.

	if processedSuccessfully, err := ProcessBlock(blocks[0], database); processedSuccessfully {
		return []*block.Block{blocks[0]}, nil
	} else {
		fmt.Printf("Error processing block: %s\n", err)
		return nil, err
	}
}

// ProcessBlock validates and adds a block to the blockchain.
func ProcessBlock(b *block.Block, database *pebble.PebbleDB) (bool, error) {
	if IsGenesisBlock(b) {
		// Handle the genesis block.
		if consensus.ValidateBlock(b, nil) {
			// Block is valid, attempt to add it to the blockchain.
			added, message := AddBlockToBlockchain(database, b)
			fmt.Println(message)
			return added, nil
		}
		// Block is invalid
		return false, fmt.Errorf("genesis block is invalid")
	}

	// Handle non-genesis blocks.
	prevBlock, err := block.DeserializeBlock(b.PrevBlock)
	if err != nil {
		return false, fmt.Errorf("error deserializing previous block: %s", err)
	}

	if consensus.ValidateBlock(b, prevBlock) {
		// Block is valid, attempt to add it to the blockchain.
		added, message := AddBlockToBlockchain(database, b)
		fmt.Println(message)
		return added, nil
	}
	// Block is invalid
	return false, fmt.Errorf("block is invalid")
}

// PrevBlockStored checks if the previous block is stored in the database.
func PrevBlockStored(b *block.Block, database *pebble.PebbleDB) (bool, error) {
	// Check if the previous block is stored in the database
	prevBlockStored, err := database.GetBlock(b.PrevBlock)
	if err != nil {
		return false, fmt.Errorf("error checking if the previous block is stored: %s", err)
	}

	if prevBlockStored == nil {
		return false, fmt.Errorf("previous block is not stored")
	}

	return true, nil
}

// CompareBlocksToBlockchain compares the blocks in the buffer to the blockchain and remove those that are already stored.
// Sort the blocks in the buffer by timestamp and return the sorted buffer.
func CompareBlocksToBlockchain(blockBuffer map[int64][]*block.Block, database *pebble.PebbleDB) map[int64][]*block.Block {
	// Iterate through the blocks in the buffer
	for timestamp, blocks := range blockBuffer {
		for _, blockk := range blocks {
			blockKey := block.ComputeHash(blockk)
			// Check if the block is already stored in the blockchain
			isStored, err := database.GetBlock(blockKey)
			if err != nil {
				fmt.Printf("Error checking if the block is stored in the blockchain: %s\n", err)
				continue
			}

			// Remove the block from the buffer if it is already stored in the blockchain
			if isStored != nil {
				delete(blockBuffer, timestamp)
			}
		}
	}

	// Sort the blocks in the buffer by timestamp
	blockSorted := sortBlocksByTimestamp(blockBuffer)

	return blockSorted
}

// sortBlocksByTimestamp sorts the blocks in the buffer by timestamp and returns the sorted buffer.
func sortBlocksByTimestamp(blockBuffer map[int64][]*block.Block) map[int64][]*block.Block {
	// Sort the blocks in the buffer by timestamp
	for timestamp, blocks := range blockBuffer {
		sort.SliceStable(blocks, func(i, j int) bool {
			return uint32(blocks[i].Timestamp) < uint32(blocks[j].Timestamp)
		})
		blockBuffer[timestamp] = blocks
	}
	return blockBuffer
}
