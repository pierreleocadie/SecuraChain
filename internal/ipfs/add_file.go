// Package storage provides functions for adding files to IPFS.
package ipfs

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	icore "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/core"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/util"
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
func AddFile(ctx context.Context, node *core.IpfsNode, ipfsApi icore.CoreAPI, filePath string) (path.ImmutablePath, error) {
	file, err := prepareFileForIPFS(filePath)
	if err != nil {
		log.Println("could not get File:", err)
	}

	fileCid, err := ipfsApi.Unixfs().Add(ctx, file)
	if err != nil {
		log.Printf("Could not add file to IPFS %v", err)
		return path.ImmutablePath{}, err
	}
	log.Printf("File added with CID: %s", fileCid.String())

	// TODO : Maybe move this to a separate function
	// Adding the file on the storage node (local system)
	home, err := os.UserHomeDir()
	if err != nil {
		return path.ImmutablePath{}, err
	}

	outputBasePath := filepath.Join(home, ".IPFS_Local_Storage/")
	if err := os.MkdirAll(outputBasePath, config.FileRights); err != nil {
		return path.ImmutablePath{}, fmt.Errorf("error creating output directory : %v", err)
	}

	outputFilePath := filepath.Join(outputBasePath, filepath.Base(filePath))

	err = util.CopyFile(filePath, outputFilePath)
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("error copying file to output directory: %v", err)
	}

	return fileCid, nil
}
