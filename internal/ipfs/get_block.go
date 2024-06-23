package ipfs

import (
	"fmt"
	"os"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// GetBlock retrieves a block from IPFS using the provided CID.
func (ipfs *IPFSNode) GetBlock(blockPath path.ImmutablePath) (block.Block, error) {
	blockFetched, err := ipfs.GetFile(blockPath)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to get the file: %v", err)
	}

	if err := files.WriteTo(blockFetched, "block"); err != nil {
		return block.Block{}, fmt.Errorf("failed to create file for the block: %v", err)
	}
	ipfs.log.Debugln("Wrote the block into a file")

	b, err := ConvertBytesToBlock(ipfs.log, "block")
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to converted the file into a *block.Block: %v", err)
	}
	ipfs.log.Debugln("Fetched CID converted to a block")

	if err := os.Remove("block"); err != nil {
		return block.Block{}, fmt.Errorf("failed to remove the file 'block': %v", err)
	}
	ipfs.log.Debugln("file removed")

	return b, err
}
