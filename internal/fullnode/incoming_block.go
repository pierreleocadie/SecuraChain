package fullnode

import (
	"fmt"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
)

// HandleIncomingBlock handles the logic for processing incoming blocks, including conflict resolution.
func HandleIncomingBlock(incomingBlock *block.Block, blockBuffer map[int64][]*block.Block, database *pebble.PebbleTransactionDB) ([]*block.Block, error) {
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
func ProcessBlock(bblock *block.Block, database *pebble.PebbleTransactionDB) (bool, error) {
	if bblock.PrevBlock == nil {
		// Handle the genesis block.
		if consensus.ValidateBlock(bblock, nil) {
			// Block is valid, attempt to add it to the blockchain.
			added, message := AddBlockToBlockchain(database, bblock)
			fmt.Println(message)
			return added, nil
		}
		// Block is invalid
		return false, fmt.Errorf("genesis block is invalid")
	}

	// Handle non-genesis blocks.
	prevBlock, err := block.DeserializeBlock(bblock.PrevBlock)
	if err != nil {
		return false, fmt.Errorf("error deserializing previous block: %s", err)
	}

	if consensus.ValidateBlock(bblock, prevBlock) {
		// Block is valid, attempt to add it to the blockchain.
		added, message := AddBlockToBlockchain(database, bblock)
		fmt.Println(message)
		return added, nil
	}
	// Block is invalid
	return false, fmt.Errorf("block is invalid")
}

func PrevBlockStored(blockk *block.Block, database *pebble.PebbleTransactionDB) (bool, err) {
	// Deserialize the previous block
	prevBlock, err := block.DeserializeBlock(blockk.PrevBlock)
	if err != nil {
		return false, fmt.Errorf("error deserializing previous block: %s", err)
	}

	isPrevBlockStored, err := database.IsIn(prevBlock)
	if err != nil {
		return false, fmt.Errorf("error checking if the previous block is stored: %s", err)
	}

	if !isPrevBlockStored {
		return false, fmt.Errorf("previous block is not stored")
	}

	return true, nil
}
