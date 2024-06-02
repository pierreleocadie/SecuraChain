// Package storage provides functions for fetching files from IPFS.
package ipfs

import (
	"context"
	"fmt"
	"log"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
)

// GetFile download a file using its CID (Content Identifier).
func GetFile(ctx context.Context, ipfsAPI icore.CoreAPI, cidFile path.ImmutablePath) (files.Node, error) {
	rootNodeFile, err := ipfsAPI.Unixfs().Get(ctx, cidFile)
	if err != nil {
		return nil, fmt.Errorf("could not get the file: %v", err)
	}
	defer rootNodeFile.Close()

	log.Printf("Got file back from IPFS (IPFS path: %s)", cidFile.String())

	return rootNodeFile, nil
}
