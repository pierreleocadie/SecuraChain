package fullnode

import (
	"fmt"
	"reflect"
	"sort"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// PrevBlockStored checks if the previous block is stored in the db.
func PrevBlockStored(log *ipfsLog.ZapEventLogger, b block.Block, db blockchaindb.BlockchainDB) error {
	prevBlockStored, err := db.GetBlock(b.PrevBlock)
	if err != nil {
		return fmt.Errorf("failed to check for previous block in db: %s", err)
	}

	if reflect.DeepEqual(prevBlockStored, block.Block{}) {
		return fmt.Errorf("previous block not found in db")
	}

	log.Debugln("Previous block found in db")
	return nil
}

// SortBlockByHeight sorts the given list of blocks by their height in ascending order.
func SortBlockByHeight(log *ipfsLog.ZapEventLogger, waitingList []*block.Block) []*block.Block {
	sort.SliceStable(waitingList, func(i, j int) bool {
		return waitingList[i].Height < waitingList[j].Height
	})

	log.Debugln("List of blocks sorted by height")
	return waitingList
}
