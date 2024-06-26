// Package storage provides functions for fetching files from IPFS.
package ipfs

import (
	"fmt"
	"log"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
)

// GetFile download a file using its CID (Content Identifier).
func (ipfs *IPFSNode) GetFile(cidFile path.ImmutablePath) (files.Node, error) {
	rootNodeFile, err := ipfs.API.Unixfs().Get(ipfs.Ctx, cidFile)
	if err != nil {
		return nil, fmt.Errorf("could not get the file: %v", err)
	}
	defer rootNodeFile.Close()

	log.Printf("Got file back from IPFS (IPFS path: %s)", cidFile.String())

	return rootNodeFile, nil
}
