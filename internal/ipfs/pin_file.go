package ipfs

import (
	"context"
	"log"

	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
)

func PinFile(ctx context.Context, ipfsAPI icore.CoreAPI, fileCid path.ImmutablePath) (bool, error) {
	if err := ipfsAPI.Pin().Add(ctx, fileCid); err != nil {
		log.Printf("Could not pin file %v", err)
		return false, err
	}

	_, IsPinned, err := ipfsAPI.Pin().IsPinned(ctx, fileCid)
	if err != nil {
		log.Printf("Could not check if file is pinned %v", err)
		return IsPinned, err
	}

	log.Printf("File pinned: %v", IsPinned)
	return IsPinned, nil
}
