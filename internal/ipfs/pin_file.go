package ipfs

import (
	"context"
	"fmt"
	"log"

	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
)

func PinFile(ctx context.Context, ipfsAPI icore.CoreAPI, fileCid path.ImmutablePath) error {
	if err := ipfsAPI.Pin().Add(ctx, fileCid); err != nil {
		log.Printf("Could not pin file %v", err)
		return fmt.Errorf("could not pin file %v", err)
	}

	_, IsPinned, err := ipfsAPI.Pin().IsPinned(ctx, fileCid)
	if err != nil {
		log.Printf("Could not check if file is pinned %v", err)
		return fmt.Errorf("could not check if file is pinned %v", err)
	}

	log.Printf("File pinned: %v", IsPinned)
	return nil
}
