// Package storage provides functions for fetching files from IPFS.
package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	icore "github.com/ipfs/boxo/coreiface"
	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
)

// RetrieveAndSaveFileByCID retrieves a file from IPFS using its CID (Content Identifier).
// It creates the necessary directory if it doesn't exist and writes the file to a specified path.
func RetrieveAndSaveFileByCID(ctx context.Context, ipfsAPI icore.CoreAPI, cidFile path.ImmutablePath) error {
	const permission = 0700

	outputBasePath := "./IPFS_Downloads"
	// Ensure the output directory exists or create it.
	if err := os.MkdirAll(outputBasePath, permission); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	rootNodeFile, err := ipfsAPI.Unixfs().Get(ctx, cidFile)
	if err != nil {
		return fmt.Errorf("could not get file with CID: %s", err)
	}

	outputPath := filepath.Join(outputBasePath, filepath.Base(cidFile.String()))

	err = files.WriteTo(rootNodeFile, outputPath)
	if err != nil {
		return fmt.Errorf("could not write out the fetched CID: %v", err)
	}

	// Print confirmation message indicating the file has been fetched and saved.
	fmt.Printf("Got file back from IPFS (IPFS path: %s) and wrote it to %s\n", cidFile.String(), outputPath)

	return nil
}
