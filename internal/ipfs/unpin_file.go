package ipfs

import (
	"fmt"
	"log"

	"github.com/ipfs/boxo/path"
)

func (ipfs *IPFSNode) UnpinFile(fileCid path.ImmutablePath) error {
	if err := ipfs.API.Pin().Rm(ipfs.Ctx, fileCid); err != nil {
		return fmt.Errorf("could not unpin file %v", err)
	}

	_, IsUnpinned, err := ipfs.API.Pin().IsPinned(ipfs.Ctx, fileCid)
	if err != nil {
		return fmt.Errorf("could not check if file is unpinned %v", err)
	}

	log.Printf("File unpinned: %v", IsUnpinned)
	return nil
}
