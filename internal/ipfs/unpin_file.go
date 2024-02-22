package ipfs

import (
	"context"
	"log"

	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
)

func UnpinFile(ctx context.Context, ipfsAPI icore.CoreAPI, fileCid path.ImmutablePath) (bool, error) {
	if err := ipfsAPI.Pin().Rm(ctx, fileCid); err != nil {
		log.Printf("Could not unpin file %v", err)
		return false, err
	}

	_, IsUnpinned, err := ipfsAPI.Pin().IsPinned(ctx, fileCid)
	if err != nil {
		log.Printf("Could not check if file is unpinned %v", err)
		return IsUnpinned, err
	}

	log.Printf("File unpinned: %v", IsUnpinned)
	return IsUnpinned, nil
}
