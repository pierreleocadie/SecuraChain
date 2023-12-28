package ipfs

import (
	"context"
	"log"

	icore "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/boxo/path"
)

func PinFile(ctx context.Context, ipfsApi icore.CoreAPI, fileCid path.ImmutablePath) (bool, error) {
	if err := ipfsApi.Pin().Add(ctx, fileCid); err != nil {
		log.Printf("Could not pin file %v", err)
		return false, err
	}

	_, IsPinned, err := ipfsApi.Pin().IsPinned(ctx, fileCid)
	if err != nil {
		log.Printf("Could not check if file is pinned %v", err)
		return IsPinned, err
	}

	log.Printf("File pinned: %v", IsPinned)
	return IsPinned, nil
}
