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

// DownloadMissingBlocks attemps to download blocks that are missing in the local blockchain from IPFS.
func DownloadMissingBlocks(log *ipfsLog.ZapEventLogger, ctx context.Context, ipfsAPI icore.CoreAPI, registryBytes []byte, db *blockchaindb.PebbleDB) (bool, []*block.Block, error) {
	var missingBlocks []*block.Block

	r, err := registry.DeserializeRegistry[registry.BlockRegistry](log, registryBytes)
	if err != nil {
		log.Errorln("Error converting bytes to block registry : ", err)
		return false, nil, fmt.Errorf("error converting bytes to block registry : %s", err)
	}
	log.Debugln("Registry converted to BlockRegistry : ", r)

	for _, blockData := range r.Blocks {
		if existingBlock, err := db.GetBlock(log, blockData.Key); err == nil && existingBlock != nil {
			log.Errorln("Block already exists in the blockchain : ", blockData.Key)
			continue
		}

		blockPath := path.FromCid(blockData.BlockCid)
		log.Debugln("Converted block CID to pathImmutable : ", blockPath.String())

		if err := ipfsAPI.Swarm().Connect(ctx, blockData.Provider); err != nil {
			log.Errorln("failed to connect to provider: %s", err)
			continue
		}
		log.Debugln("Connected to provider %s", blockData.Provider.ID)

		downloadBlock, err := ipfs.GetBlock(log, ctx, ipfsAPI, blockPath)
		if err != nil {
			log.Errorln("Error downloading block from IPFS : ", err)
			return false, nil, fmt.Errorf("error downloading block from IPFS : %s", err)
		}
		log.Debugln("Block downloaded from IPFS : ", downloadBlock)

		missingBlocks = append(missingBlocks, downloadBlock)
	}

	log.Debugln("Number of missing blocks donwnloaded : ", len(missingBlocks))
	return true, missingBlocks, nil
}

func GetMissingBlocks(log *ipfsLog.ZapEventLogger, ctx context.Context, r registry.BlockRegistry, db *blockchaindb.PebbleDB) []*registry.BlockData {
	var missingBlocks []*registry.BlockData

	for _, blockData := range r.Blocks {

		if existingBlock, err := db.GetBlock(log, blockData.Key); err == nil && existingBlock != nil {
			log.Errorln("Block already exists in the blockchain : ", blockData.Key)
			continue
		}

		missingBlocks = append(missingBlocks, &blockData)
	}

	log.Debugln("Number of missing blocks : ", len(missingBlocks))
	return missingBlocks
}
