package ipfs

import (
	"context"
	"fmt"
	"os"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/registry"
)

// GetBlock retrieves a block from IPFS using the provided CID.
func GetBlock(log *ipfsLog.ZapEventLogger, ctx context.Context, ipfsAPI icore.CoreAPI, blockPath path.ImmutablePath) (*block.Block, error) {
	blockFetched, err := GetFile(ctx, ipfsAPI, blockPath)
	if err != nil {
		log.Errorln("Failed to get the file: %s ", err)
		return nil, fmt.Errorf("failed to get the file: %v", err)
	}

	if err := files.WriteTo(blockFetched, "block"); err != nil {
		log.Errorln("Failed to write the block into a file: %s ", err)
		return nil, fmt.Errorf("failed to create file for the block: %v", err)
	}
	log.Debugln("Wrote the block into a file")

	b, err := registry.ConvertToBlock(log, "block")
	if err != nil {
		log.Errorln("Failed to convert the file into a *block.Block: %s ", err)
		return nil, fmt.Errorf("failed to converted the file into a *block.Block: %v", err)
	}
	log.Debugln("Converted the fetched CID to a block")

	if err := os.Remove("block"); err != nil {
		log.Errorln("Failed to remove the file 'block': %s ", err)
		return nil, fmt.Errorf("failed to remove the file 'block': %v", err)
	}
	log.Debugln("Removed the file")

	return b, err
}
