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
func AddFileToIPFS(ctx context.Context, node *core.IpfsNode, ipfsApi icore.CoreAPI, inputPathFile string) (path.ImmutablePath, string, error) {
	someFile, err := prepareFileForIPFS(inputPathFile)
	if err != nil {
		log.Println("could not get File:", err)
	}

	cidFile, err := ipfsApi.Unixfs().Add(ctx, someFile)
	if err != nil {
		log.Println("could not add File: ", err)
	}
	fmt.Printf("Added file to IPFS with CID %s\n", cidFile.String())

	// Pin a file with IPFS
	if err := ipfsApi.Pin().Add(ctx, cidFile); err != nil {
		log.Println("Could not pin the file in IPFS network ", err)
	}
	_, IsPinned, err := ipfsApi.Pin().IsPinned(ctx, cidFile)
	if err != nil {
		return path.ImmutablePath{}, "", err
	}

	fmt.Println("Le fichier a bien été pin avec le cid: ", IsPinned)

	// Collect file information and metadata.
	fileName, fileSize, fileType, err := utils.FileInfo(inputPathFile)

	if err != nil {
		log.Fatal(err)
	}

	// Structure for storing file metadata.
	data := util.FileMetaData{
		Cid:           cidFile.String(),
		Timestamp:     time.Now(),
		FileSize:      fileSize,
		Extension:     fileType,
		OriginalName:  fileName,
		UserPublicKey: node.Identity.String(),
	}

	// Check if the file exists
	fileNameJSON := "ipfs_file_storage.json"

	var storage = util.CIDStorage{}
	if _, err := os.Stat(fileNameJSON); os.IsNotExist(err) {
		storage.Files = append(storage.Files, data)
		// Save the metadata to a JSON file.
		if err := util.SaveToJSON(fileNameJSON, storage); err != nil {
			log.Fatalf("Error saving JSON data %v", err)
		}
	} else {
		// File exists, load existing data
		fileData, err := util.LoadFromJSON(fileNameJSON)
		if err != nil {
			return path.ImmutablePath{}, "", err
		}
		// storage.Files = append(storage.Files, fileData)
		fileData.Files = append(fileData.Files, data)
		fmt.Println("file data is ", fileData)

		// Save the metadata to a JSON file.
		if err := util.SaveToJSON(fileNameJSON, fileData); err != nil {
			log.Fatalf("Error saving JSON data %v", err)
		}
	}
	// Adding the file on the storage node (local system)
	home, err := os.UserHomeDir()
	if err != nil {
		return path.ImmutablePath{}, "", err
	}

	outputBasePath := filepath.Join(home, ".IPFS_Local_Storage/")
	if err := os.MkdirAll(outputBasePath, config.FileRights); err != nil {
		return path.ImmutablePath{}, "", fmt.Errorf("error creating output directory : %v", err)
	}

	outputFilePath := filepath.Join(outputBasePath, filepath.Base(inputPathFile))

	err = util.CopyFile(inputPathFile, outputFilePath)
	if err != nil {
		return path.ImmutablePath{}, "", fmt.Errorf("error copying file to output directory: %v", err)
	}

	return cidFile, fileName, nil
}
