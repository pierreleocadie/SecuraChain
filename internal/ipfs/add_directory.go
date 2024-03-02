// Package storage provides functions for adding files to IPFS.
package ipfs

import (
	"context"
	"log"

	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

// AddFileToIPFS adds a file to IPFS and returns its CID. It also collects and saves file metadata.
// The function handles the file addition process and records metadata such as file size, type, name, and user public key.
func AddDirectory(ctx context.Context, config *config.Config, ipfsAPI icore.CoreAPI, directoryPath string) (path.ImmutablePath, error) {
	directory, err := PrepareFileForIPFS(directoryPath)
	if err != nil {
		log.Println("could not get File:", err)
	}

	directoryCid, err := ipfsAPI.Unixfs().Add(ctx, directory)
	if err != nil {
		log.Printf("Could not add file to IPFS %v", err)
		return path.ImmutablePath{}, err
	}
	log.Printf("Directory added with CID: %s", directoryCid.String())

	return directoryCid, nil
}
