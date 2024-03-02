// Package storage provides functions for adding files to IPFS.
package ipfs

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

// AddFileToIPFS adds a file to IPFS and returns its CID. It also collects and saves file metadata.
// The function handles the file addition process and records metadata such as file size, type, name, and user public key.
func AddFile(ctx context.Context, config *config.Config, ipfsAPI icore.CoreAPI, filePath string) (path.ImmutablePath, error) {
	file, err := PrepareFileForIPFS(filePath)
	if err != nil {
		log.Println("could not get File:", err)
	}

	fileCid, err := ipfsAPI.Unixfs().Add(ctx, file)
	if err != nil {
		log.Printf("Could not add file to IPFS %v", err)
		return path.ImmutablePath{}, err
	}
	log.Printf("File added with CID: %s", fileCid.String())

	// Adding the file on the storage node (local system)
	home, err := os.UserHomeDir()
	if err != nil {
		return path.ImmutablePath{}, err
	}

	localStoragePath := filepath.Join(home, ".IPFS_Local_Storage/")
	if err := os.MkdirAll(localStoragePath, os.FileMode(config.FileRights)); err != nil {
		return path.ImmutablePath{}, fmt.Errorf("error creating output directory : %v", err)
	}

	outputFilePath := filepath.Join(localStoragePath, filepath.Base(filePath))

	err = MoveFile(filePath, outputFilePath)
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("error copying file to output directory: %v", err)
	}

	// restore file permissions
	if err := os.Chmod(outputFilePath, os.FileMode(config.FileRights)); err != nil {
		return path.ImmutablePath{}, fmt.Errorf("error restoring file permissions: %v", err)
	}

	return fileCid, nil
}
