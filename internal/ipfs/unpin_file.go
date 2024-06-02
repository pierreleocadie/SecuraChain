package ipfs

import (
	"context"
	"fmt"
	"log"

	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
)

func UnpinFile(ctx context.Context, ipfsAPI icore.CoreAPI, fileCid path.ImmutablePath) error {
	if err := ipfsAPI.Pin().Rm(ctx, fileCid); err != nil {
		log.Printf("Could not unpin file %v", err)
		return fmt.Errorf("could not unpin file %v", err)
	}

	_, IsUnpinned, err := ipfsAPI.Pin().IsPinned(ctx, fileCid)
	if err != nil {
		log.Printf("Could not check if file is unpinned %v", err)
		return fmt.Errorf("could not check if file is unpinned %v", err)
	}

	log.Printf("File unpinned: %v", IsUnpinned)
	return nil
}
