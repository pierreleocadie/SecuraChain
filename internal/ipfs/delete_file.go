package ipfs

import (
	"log"
	"os"
	"path/filepath"
)

// This function allows to delete a file from IPFS, and from the storer node (precisely unpin it)
func DeleteFile(fileName string) error {
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

	return nil
}
