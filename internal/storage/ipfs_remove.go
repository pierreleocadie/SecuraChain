package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	icore "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/boxo/path"
	"github.com/pierreleocadie/SecuraChain/internal/util"
)

// This function allows to delete a file from IPFS, and from the storer node (precisely unpin it)
func DeleteFromIPFS(ctx context.Context, ipfsAPI icore.CoreAPI, cid path.ImmutablePath, fileName string) error {
	// Unpin the file on IPFS
	if err := ipfsAPI.Pin().Rm(ctx, cid); err != nil {
		log.Println("Failed to unpin the file from IPFS", err)
	}
	_, IsUnPinned, err := ipfsAPI.Pin().IsPinned(ctx, cid)
	if err != nil {
		return err
	}

	fmt.Println("The file has been unpined with success: ", IsUnPinned)

	// Deleting the file on the storage node (local system)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err != nil {
		log.Fatal(err)
	}

	outputBasePath := filepath.Join(home, ".IPFS_Local_Storage/")
	outputBaseFile := filepath.Join(outputBasePath, fileName)
	if err := os.Remove(outputBaseFile); err != nil {
		return err
	}

	// Read the JSON File
	fileNameJSON := "ipfs_file_storage.json"
	fileData, err := util.LoadFromJSON(fileNameJSON)

	if err != nil {
		return err
	}
	// Find and delete the metadata
	for i, file := range fileData.Files {
		if file.OriginalName == fileName {
			fileData.Files = append(fileData.Files[:i], fileData.Files[i+1:]...)
			break
		}
	}

	// Save the new metadata
	if err := util.SaveToJSON(fileNameJSON, fileData); err != nil {
		log.Fatalf("Error saving JSON data: %v", err)
	}

	fmt.Println("Métadonnée supprimée avec succès")
	return nil
}
