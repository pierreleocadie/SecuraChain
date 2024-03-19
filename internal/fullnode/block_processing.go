package fullnode

import (
	"fmt"
	"sort"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
)

// ProcessBlock validates block and adds a given block to the blockchain.
func ProcessBlock(log *ipfsLog.ZapEventLogger, b *block.Block, database *blockchaindb.BlockchainDB) (bool, error) {
	// Handle the genesis block.
	if block.IsGenesisBlock(b) {
		if consensus.ValidateBlock(b, nil) {
			log.Debugln("Genesis block validation succeeded")
			added, message := blockchaindb.AddBlockToBlockchain(b, database)
			log.Debugln(message)
			return added, nil
		}
		return false, fmt.Errorf("genesis block validation failed")
	}

	// Handle non-genesis blocks.
	prevBlock, err := database.GetBlock(b.PrevBlock)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve previous block: %s", err)
	}

	if consensus.ValidateBlock(b, prevBlock) {
		log.Debugln("Block validation succeeded")
		added, message := blockchaindb.AddBlockToBlockchain(b, database)
		log.Debugln(message)
		return added, nil
	}
	return false, fmt.Errorf("block validation failed")
}

// PrevBlockStored checks if the previous block is stored in the database.
func PrevBlockStored(log *ipfsLog.ZapEventLogger, b *block.Block, database *blockchaindb.BlockchainDB) (bool, error) {
	prevBlockStored, err := database.GetBlock(b.PrevBlock)
	if err != nil {
		return false, fmt.Errorf("failed to check for previous block in database: %s", err)
	}

	if prevBlockStored == nil {
		log.Debugln("Previous block not found in database")
		return false, nil
	}

	log.Debugln("Previous block found in database")
	return true, nil
}

// SortBlockByHeight sorts the given list of blocks by their height in ascending order.
func SortBlockByHeight(log *ipfsLog.ZapEventLogger, waitingList []*block.Block) []*block.Block {
	sort.SliceStable(waitingList, func(i, j int) bool {
		return waitingList[i].Height < waitingList[j].Height
	})

	log.Debugln("List of blocks sorted by height")
	return waitingList
}
