// Package storage provides functions for fetching files from IPFS.
package ipfs

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

// GetFile download a file using its CID (Content Identifier).
// It creates the necessary directory if it doesn't exist and writes the file to a specified path.
func GetFile(ctx context.Context, config *config.Config, ipfsAPI icore.CoreAPI, cidFile path.ImmutablePath) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	downloadsStoragePath := filepath.Join(home, ".IPFS_Downloads/")
	// Ensure the output directory exists or create it.
	if err := os.MkdirAll(downloadsStoragePath, os.FileMode(config.FileRights)); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	log.Printf("Getting file with CID: %s\n", cidFile.String())
	rootNodeFile, err := ipfsAPI.Unixfs().Get(ctx, cidFile)
	if err != nil {
		return fmt.Errorf("could not get file with CID: %s", err)
	}
	defer rootNodeFile.Close()
	log.Printf("Got the file")

	downloadedFilePath := filepath.Join(downloadsStoragePath, filepath.Base(cidFile.String()))

	log.Printf("Writing file to %s\n", downloadedFilePath)
	err = files.WriteTo(rootNodeFile, downloadedFilePath)
	if err != nil {
		return fmt.Errorf("could not write out the fetched CID: %v", err) // send error to channel
	}

	// Print confirmation message indicating the file has been fetched and saved.
	log.Printf("Got file back from IPFS (IPFS path: %s) and wrote it to %s\n", cidFile.String(), downloadedFilePath)

	return nil
}

// GetDirectoryWithPath download a directory using its CID (Content Identifier).
// It creates the necessary directory if it doesn't exist and writes the file to a specified path.
func GetDirectoryWithPath(ctx context.Context, ipfsAPI icore.CoreAPI, directoryFile path.ImmutablePath, downloadPath string) error {
	local, err := os.Getwd()
	if err != nil {
		return err
	}

	downloadsStoragePath := filepath.Join(local, downloadPath)
	// Ensure the output directory exists or create it.
	if err := os.MkdirAll(downloadsStoragePath, config.FileRights); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	log.Printf("Getting file with CID: %s\n", directoryFile.String())

	rootNodeFile, err := ipfsAPI.Unixfs().Get(ctx, directoryFile)
	if err != nil {
		return fmt.Errorf("could not get directory with CID: %s", err)
	}
	defer rootNodeFile.Close()
	log.Printf("Got the directory")

	downloadedFilePath := filepath.Join(downloadPath, filepath.Base(directoryFile.String()))

	var wg sync.WaitGroup
	errCh := make(chan error, 1) // create an error channel
	log.Printf("Writing file to %s\n", downloadedFilePath)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = files.WriteTo(rootNodeFile, downloadedFilePath)
		if err != nil {
			errCh <- fmt.Errorf("could not write out the fetched CID: %v", err) // send error to channel
			return
		}
	}()

	wg.Wait()

	close(errCh) // close the channel

	// check if there was an error
	if err := <-errCh; err != nil {
		fmt.Println(err)
	}

	// Print confirmation message indicating the directory has been fetched and saved.
	log.Printf("Got directory back from IPFS (IPFS path: %s) and wrote it to %s\n", directoryFile.String(), downloadedFilePath)

	return nil
}
