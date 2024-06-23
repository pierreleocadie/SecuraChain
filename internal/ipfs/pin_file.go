package ipfs

import (
	"fmt"
	"log"

	"github.com/ipfs/boxo/path"
)

func (ipfs *IPFSNode) PinFile(fileCid path.ImmutablePath) error {
	if err := ipfs.API.Pin().Add(ipfs.Ctx, fileCid); err != nil {
		return fmt.Errorf("could not pin file %v", err)
	}

	_, IsPinned, err := ipfs.API.Pin().IsPinned(ipfs.Ctx, fileCid)
	if err != nil {
		return fmt.Errorf("could not check if file is pinned %v", err)
	}

	log.Printf("File pinned: %v", IsPinned)
	return nil
}
