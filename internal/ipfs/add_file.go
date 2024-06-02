// Package storage provides functions for adding files to IPFS.
package ipfs

import (
	"context"
	"fmt"
	"log"

	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

// AddFile adds a file to IPFS and returns the CID of the file.
func AddFile(ctx context.Context, config *config.Config, ipfsAPI icore.CoreAPI, filePath string) (path.ImmutablePath, error) {
	file, err := PrepareFileForIPFS(filePath)
	if err != nil {
		log.Println("could not get File:", err)
		return path.ImmutablePath{}, fmt.Errorf("could not get File: %v", err)
	}

	fileCid, err := ipfsAPI.Unixfs().Add(ctx, file)
	if err != nil {
		log.Printf("Could not add file to IPFS %v", err)
		return path.ImmutablePath{}, fmt.Errorf("could not add file to IPFS %v", err)
	}

	log.Printf("File added to IPFS with CID: %s", fileCid.String())
	return fileCid, nil
}
