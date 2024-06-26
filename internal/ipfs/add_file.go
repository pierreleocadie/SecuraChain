// Package storage provides functions for adding files to IPFS.
package ipfs

import (
	"fmt"
	"log"

	"github.com/ipfs/boxo/path"
)

// AddFile adds a file to IPFS and returns the CID of the file.
func (ipfs *IPFSNode) AddFile(filePath string) (path.ImmutablePath, error) {
	file, err := ipfs.prepareFileForIPFS(filePath)
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("could not get File: %v", err)
	}

	fileCid, err := ipfs.API.Unixfs().Add(ipfs.Ctx, file)
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("could not add file to IPFS %v", err)
	}

	log.Printf("File added to IPFS with CID: %s", fileCid.String())
	return fileCid, nil
}
