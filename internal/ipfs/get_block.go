package ipfs

import (
	"context"
	"fmt"
	"os"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
)

func GetBlock(ctx context.Context, config *config.Config, ipfsAPI icore.CoreAPI, cidBlock string, blockName string) (*block.Block, error) {

	cid, err := path.NewPath(cidBlock)
	if err != nil {
		return nil, fmt.Errorf("could not parse CID to path: %s", err)
	}

	rootNode, err := ipfsAPI.Unixfs().Get(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("could not get directory with CID: %s", err)
	}

	err = files.WriteTo(rootNode, blockName)
	if err != nil {
		return nil, fmt.Errorf("could not write out the fetched CID: %v", err)
	}

	b, err := pebble.ConvertToBlock(blockName)
	if err != nil {
		return nil, fmt.Errorf("could not convert the fetched CID to a block: %v", err)
	}

	os.Remove(blockName)

	return b, err
}
