// Package storage provides functions for adding files to IPFS.
package ipfs

import (
	"context"
	"fmt"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// AddFileToIPFS adds a file to IPFS and returns its CID. It also collects and saves file metadata.
// The function handles the file addition process and records metadata such as file size, type, name, and user public key.
func AddBlockToIPFS(ctx context.Context, config *config.Config, ipfsAPI icore.CoreAPI, b *block.Block) (path.ImmutablePath, error) {
	blockBytes, err := b.Serialize()
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("could not serialize block: %s", err)
	}

	peerCidFile, err := ipfsAPI.Unixfs().Add(ctx,
		files.NewBytesFile(blockBytes))
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("could not add File: %s", err)
	}

	fmt.Printf("Added file to peer with CID %s\n", peerCidFile.String())

	return peerCidFile, nil

}
