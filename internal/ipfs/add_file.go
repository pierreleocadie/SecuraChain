// Package storage provides functions for adding files to IPFS.
package ipfs

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

// prepareFileForIPFS prepares a file to be added to IPFS by creating a UnixFS node from the given path.
// It retrieves file information and creates a serial file node for IPFS.
func prepareFileForIPFS(path string) (files.Node, error) {
	sanitizedPath, err := utils.SanitizePath(path)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(sanitizedPath)
	if err != nil {
		return nil, err
	}

	fileNode, err := files.NewSerialFile(path, false, stat)
	if err != nil {
		return nil, err
	}

	return fileNode, nil
}

// AddFileToIPFS adds a file to IPFS and returns its CID. It also collects and saves file metadata.
// The function handles the file addition process and records metadata such as file size, type, name, and user public key.
func AddFile(ctx context.Context, ipfsAPI icore.CoreAPI, filePath string) (path.ImmutablePath, error) {
	file, err := prepareFileForIPFS(filePath)
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
	if err := os.MkdirAll(localStoragePath, config.FileRights); err != nil {
		return path.ImmutablePath{}, fmt.Errorf("error creating output directory : %v", err)
	}

	outputFilePath := filepath.Join(localStoragePath, filepath.Base(filePath))

	err = MoveFile(filePath, outputFilePath)
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("error copying file to output directory: %v", err)
	}

	// restore file permissions
	if err := os.Chmod(outputFilePath, config.FileRights); err != nil {
		return path.ImmutablePath{}, fmt.Errorf("error restoring file permissions: %v", err)
	}

	return fileCid, nil
}
