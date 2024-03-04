package fullnode

import (
	"fmt"
	"sort"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/pebble"
)

// // HandleIncomingBlock handles the logic for processing incoming blocks, including conflict resolution.
// func HandleIncomingBlock(incomingBlock *block.Block, blockBuffer map[int64]*block.Block, database *pebble.PebbleDB) (map[int64]*block.Block, error) {
// 	var timeToWait = 5 * time.Second

// 	// Add the block to the buffer based on its timestamp.
// 	timestamp := incomingBlock.Timestamp
// 	blockBuffer[timestamp] = incomingBlock

// 	for time.Sleep(timeToWait) {
// 		blockBuffer[]

// 	}

// }

// IsGenesisBlock checks if the block is the genesis block
func IsGenesisBlock(b *block.Block) bool {
	if b.PrevBlock == nil && b.Header.Height == 1 {
		return true
	}
	return false
}

// ProcessBlock validates and adds a block to the blockchain.
func ProcessBlock(b *block.Block, database *pebble.PebbleDB) (bool, error) {
	if IsGenesisBlock(b) {
		// Handle the genesis block.
		if consensus.ValidateBlock(b, nil) {
			// Block is valid, attempt to add it to the blockchain.
			added, message := pebble.AddBlockToBlockchain(b, database)
			fmt.Println(message)
			return added, nil
		}
		// Block is invalid
		return false, fmt.Errorf("genesis block is invalid")
	}

	// Handle non-genesis blocks.
	prevBlock, err := database.GetBlock(b.PrevBlock)
	if err != nil {
		return false, fmt.Errorf("error getting previous block: %s", err)
	}

	if consensus.ValidateBlock(b, prevBlock) {
		// Block is valid, attempt to add it to the blockchain.
		added, message := pebble.AddBlockToBlockchain(b, database)
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
func CompareBlocksToBlockchain(blockBuffer map[int64]*block.Block, database *pebble.PebbleDB) []*block.Block {
	// Iterate through the blocks in the buffer
	for timestamp, blocks := range blockBuffer {
		blockKey := block.ComputeHash(blocks)
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

	// Sort the blocks in the buffer by timestamp
	blockSorted := sortBlocksByTimestamp(blockBuffer)

	return blockSorted
}

// sortBlocksByTimestamp sorts the blocks in the buffer by timestamp and returns the sorted buffer.
func sortBlocksByTimestamp(blockBuffer map[int64]*block.Block) []*block.Block {
	var blocks []*block.Block

	for _, block := range blockBuffer {
		blocks = append(blocks, block)
	}

	// Sort the blocks in the buffer by timestamp
	sort.SliceStable(blocks, func(i, j int) bool {
		return blocks[i].Timestamp < blocks[j].Timestamp
	})

	return blocks
}
