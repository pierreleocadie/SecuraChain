package fullnode

import (
	"fmt"
	"sort"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/registry"
)

// PrevBlockStored checks if the previous block is stored in the db.
func PrevBlockStored(log *ipfsLog.ZapEventLogger, b *block.Block, db *blockchaindb.BlockchainDB) (bool, error) {
	prevBlockStored, err := db.GetBlock(log, b.PrevBlock)
	if err != nil {
		log.Errorln("Failed to check for previous block in db: ", err)
		return false, fmt.Errorf("failed to check for previous block in db: %s", err)
	}

	if prevBlockStored == nil {
		log.Warn("Previous block not found in db")
		return false, nil
	}

	log.Debugln("Previous block found in db")
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

// AddBlockTransactionToRegistry adds the transactions of the given block to the indexing registry.
func AddBlockTransactionToRegistry(log *ipfsLog.ZapEventLogger, config *config.Config, b *block.Block) bool {
	for _, tx := range b.Transactions {
		switch tx.(type) {
		case *transaction.AddFileTransaction:
			addFileTransac := tx.(*transaction.AddFileTransaction)
			if err := registry.AddFileToRegistry(log, config, addFileTransac); err != nil {
				log.Errorln("Error adding file to registry: ", err)
				return false
			}

			log.Debugln("[AddBlockTransactionToRegistry] - File added to the registry")

		case *transaction.DeleteFileTransaction:
			deleteFileTransac := tx.(*transaction.DeleteFileTransaction)
			if err := registry.DeleteFileFromRegistry(log, config, deleteFileTransac); err != nil {
				log.Errorln("Error deleting file from registry: ", err)
				return false
			}

			log.Debugln("[AddBlockTransactionToRegistry] - File deleted from the registry")
		}
	}

	log.Debugln("Transactions of block updated to the registry")
	return true
}
