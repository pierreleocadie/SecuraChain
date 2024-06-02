package fullnode

import (
	"context"
	"fmt"

	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/registry"
)

// GetMissingBlocks returns a list of missing blocks by comparing the blocks in the given BlockRegistry
// with the blocks stored in the PebbleDB. It checks if each block exists in the database and if not,
// adds it to the list of missing blocks.
func GetMissingBlocks(log *ipfsLog.ZapEventLogger, r registry.BlockRegistry, db *blockchaindb.PebbleDB) []*registry.BlockData {
	var missingBlocks []*registry.BlockData

	for _, blockData := range r.Blocks {

		if existingBlock, err := db.GetBlock(log, blockData.Key); err == nil && existingBlock != nil {
			log.Errorln("Block already exists in the blockchain : ", blockData.Key)
			continue
		}

		missingBlocks = append(missingBlocks, &blockData)
	}

	log.Infoln("Number of missing blocks : ", len(missingBlocks))
	return missingBlocks
}

// DownloadMissingBlocks attemps to download blocks that are missing in the local blockchain from IPFS.
func DownloadMissingBlocks(log *ipfsLog.ZapEventLogger, ctx context.Context, ipfsAPI icore.CoreAPI, missingBlocks []*registry.BlockData) ([]*block.Block, error) {
	var downloadedBlocks []*block.Block

	for _, blockData := range missingBlocks {
		blockPath := path.FromCid(blockData.BlockCid)
		log.Debugln("Converted block CID to pathImmutable : ", blockPath.String())

		if err := ipfsAPI.Swarm().Connect(ctx, blockData.Provider); err != nil {
			log.Errorln("failed to connect to provider: %s", err)
			continue
		}
		log.Debugln("Connected to provider %s", blockData.Provider.ID)

		b, err := ipfs.GetBlock(log, ctx, ipfsAPI, blockPath)
		if err != nil {
			log.Errorln("Error downloading block from IPFS : ", err)
			return nil, fmt.Errorf("error downloading block from IPFS : %s", err)
		}
		log.Debugln("Block downloaded from IPFS : ", b)
		downloadedBlocks = append(downloadedBlocks, b)

	}

	return downloadedBlocks, nil
}
