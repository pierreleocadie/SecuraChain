package blockchaindb

import (
	"context"
	"fmt"
	"os"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

func GetBlockFromIPFS(ctx context.Context, ipfsAPI icore.CoreAPI, cidBlock path.ImmutablePath) (*block.Block, error) {

	// cid, err := path.NewPath(cidBlock)
	// if err != nil {
	// 	return nil, fmt.Errorf("could not parse CID to path: %s", err)
	// }

	rootNode, err := ipfsAPI.Unixfs().Get(ctx, cidBlock)
	if err != nil {
		return nil, fmt.Errorf("could not get directory with CID: %s", err)
	}

	err = files.WriteTo(rootNode, "block")
	if err != nil {
		return nil, fmt.Errorf("could not write out the fetched CID: %v", err)
	}

	b, err := ConvertToBlock("block")
	if err != nil {
		return nil, fmt.Errorf("could not convert the fetched CID to a block: %v", err)
	}

	if err := os.Remove("block"); err != nil {
		return nil, fmt.Errorf("could not remove the file: %v", err)
	}

	return b, err
}
