package ipfs

import (
	"context"
	"fmt"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

func GetDirectory(ctx context.Context, config *config.Config, ipfsAPI icore.CoreAPI, cidDirectory path.ImmutablePath, blockchainName string) error {

	rootNodeDirectory, err := ipfsAPI.Unixfs().Get(ctx, cidDirectory)
	if err != nil {
		return fmt.Errorf("could not get directory with CID: %s", err)
	}

	err = files.WriteTo(rootNodeDirectory, blockchainName)
	if err != nil {
		return fmt.Errorf("could not write out the fetched CID: %v", err)
	}

	return nil
}
