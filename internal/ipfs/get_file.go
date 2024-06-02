// Package storage provides functions for fetching files from IPFS.
package ipfs

import (
	"context"
	"fmt"
	"log"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

// GetFile download a file using its CID (Content Identifier).
func GetFile(ctx context.Context, config *config.Config, ipfsAPI icore.CoreAPI, cidFile path.ImmutablePath) (files.Node, error) {
	log.Printf("Getting file with CID: %s\n", cidFile.String())
	rootNodeFile, err := ipfsAPI.Unixfs().Get(ctx, cidFile)
	if err != nil {
		return nil, fmt.Errorf("could not get the file: %v", err)
	}
	defer rootNodeFile.Close()

	log.Printf("Got file back from IPFS (IPFS path: %s)", cidFile.String())

	return rootNodeFile, nil
}
