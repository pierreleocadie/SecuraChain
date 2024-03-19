package blockchaindb

import (
	"context"
	"fmt"
	"os"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// GetBlockFromIPFS retrieves a block from IPFS using the provided CID.
func GetBlockFromIPFS(log *ipfsLog.ZapEventLogger, ctx context.Context, ipfsAPI icore.CoreAPI, blockPath path.ImmutablePath) (*block.Block, error) {
	blockFetched, err := ipfsAPI.Unixfs().Get(ctx, blockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the block from IPFS %s", err)
	}
	log.Debugln("Retrieved block from IPFS: %s ", blockPath.String())

	if err := files.WriteTo(blockFetched, "block"); err != nil {
		return nil, fmt.Errorf("failed to create file for the block: %v", err)
	}
	log.Debugln("Wrote the block into a file")

	b, err := ConvertToBlock("block")
	if err != nil {
		return nil, fmt.Errorf("failed to converted the file into a *block.Block: %v", err)
	}
	log.Debugln("Converted the fetched CID to a block")

	if err := os.Remove("block"); err != nil {
		return nil, fmt.Errorf("failed to remove the file 'block': %v", err)
	}
	log.Debugln("Removed the file")

	return b, err
}
