package fullnode

import (
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
)

func AddBlockToBlockchain(database *pebble.PebbleTransactionDB, blockAnnounced *block.Block) (bool, string) {

	key := []byte(blockAnnounced.Signature)
	existingBlock, err := database.GetBlock(key)
	if err == nil && existingBlock != nil {
		return false, "Block already existing in the blockchain"
	}

	//if the block is not in the blockchain, add it
	err = database.SaveBlock(key, blockAnnounced)
	if err != nil {
		return false, "Error adding block to the database"
	} else {
		return true, "Block addded to the blockchain succesfully"
	}
}
