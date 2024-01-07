package ipfs

import (
	"context"
	"log"

	icore "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/boxo/path"
)

func UnpinFile(ctx context.Context, ipfsApi icore.CoreAPI, fileCid path.ImmutablePath) (bool, error) {
	if err := ipfsApi.Pin().Rm(ctx, fileCid); err != nil {
		log.Printf("Could not unpin file %v", err)
		return false, err
	}

	_, IsUnpinned, err := ipfsApi.Pin().IsPinned(ctx, fileCid)
	if err != nil {
		log.Printf("Could not check if file is unpinned %v", err)
		return IsUnpinned, err
	}

	log.Printf("File unpinned: %v", IsUnpinned)
	return IsUnpinned, nil
}