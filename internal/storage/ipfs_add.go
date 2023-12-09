// Package storage provides functions for adding files to IPFS.
package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	icore "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/core"
	"github.com/pierreleocadie/SecuraChain/internal/util"
)

// PrepareFileForIPFS prepares a file to be added to IPFS by creating a UnixFS node from the given path.
// It retrieves file information and creates a serial file node for IPFS.
func PrepareFileForIPFS(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// AddFileToIPFS adds a file to IPFS and returns its CID. It also collects and saves file metadata.
// The function handles the file addition process and records metadata such as file size, type, name, and user public key.
func AddFileToIPFS(ctx context.Context, node *core.IpfsNode, ipfsApi icore.CoreAPI, inputPathFile string) (path.ImmutablePath, error) {
	someFile, err := PrepareFileForIPFS(inputPathFile)
	if err != nil {
		fmt.Errorf("could not get File: %s", err)
	}

	cidFile, err := ipfsApi.Unixfs().Add(ctx, someFile)
	if err != nil {
		fmt.Errorf("could not add File: %s", err)
	}
	fmt.Printf("Added file to IPFS with CID %s\n", cidFile.String())

	// Pin a file with IPFS
	ipfsApi.Pin().Add(ctx, cidFile)
	_, IsPinned, err := ipfsApi.Pin().IsPinned(ctx, cidFile)
	if err != nil {
		return path.ImmutablePath{}, err
	}

	fmt.Println("Le fichier a bien été pin avec le cid: ", IsPinned)

	// Collect file information and metadata.
	fileName, fileSize, fileType, err := FileInfo(inputPathFile)

	if err != nil {
		log.Fatal(err)
	}

	// Structure for storing file metadata.
	data := util.FileMetaData{
		Cid:           cidFile.String(),
		Timestamp:     time.Now(),
		FileSize:      fileSize,
		FileType:      fileType,
		OriginalName:  fileName,
		UserPublicKey: node.Identity.String(),
	}

	// Create a storage structure for the file metadata.
	datav1 := util.CIDStorage{}
	datav1.Files = append(datav1.Files, data)

	// Save the metadata to a JSON file.
	if err := util.SaveToJSON("ipfs_file_storage.json", datav1); err != nil {
		log.Fatalf("Error saving JSON data %v", err)

	}

	// Adding the file on the storage node (local system)
	home, err := os.UserHomeDir()
	if err != nil {
		return path.ImmutablePath{}, err
	}

	outputBasePath := filepath.Join(home, ".IPFS_Local_Storage/")
	if err := os.MkdirAll(outputBasePath, 0700); err != nil {
		return path.ImmutablePath{}, fmt.Errorf("error creating output directory : %v", err)
	}

	outputFilePath := filepath.Join(outputBasePath, filepath.Base(inputPathFile))

	err = util.CopyFile(inputPathFile, outputFilePath)
	if err != nil {
		return path.ImmutablePath{}, fmt.Errorf("error copying file to output directory: %v", err)
	}

	return cidFile, nil
}
