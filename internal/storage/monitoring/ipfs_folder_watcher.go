// Package monitoring provides functionality to watch a specific folder for file changes and interact with IPFS accordingly.
package monitoring

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	icore "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/core"
	"github.com/pierreleocadie/SecuraChain/internal/storage"
)

const permission = 0700

// CreateStorageQueueDirectory creates a directory for adding files to the storage node.
func CreateStorageQueueDirectory() error {
	outputBasePath := "./Storage_Queue"
	// S'assurez que le dossier de sortie existe
	if err := os.MkdirAll(outputBasePath, permission); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}
	fmt.Println("Ajoutez vos fichiers dans Storage_Queue/")
	return nil
}

// MonitorinRepoInit watches a directory for new files and adds them to IPFS.
// It initializes the directory for monitoring, creates a watcher, and processes file events.
func WatchStorageQueueForChanges(ctx context.Context, node *core.IpfsNode, ipfsAPI icore.CoreAPI) (path.ImmutablePath, string, error) {
	if err := CreateStorageQueueDirectory(); err != nil {
		log.Fatal(err)
	}
	watchDir := "./Storage_Queue"

	// Create a new fsnotify watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	cidChan := make(chan path.ImmutablePath) // Channel for passing the CID
	fileNameChan := make(chan string)        // Channel for passing the the filename of the file

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					fmt.Println("Error")
				}
				// Check the type of event and log the details.
				fmt.Printf("event: %v\n", event)

				if event.Op&fsnotify.Create == fsnotify.Create {
					fmt.Printf("created file: %s\n", event.Name)
					cidFile, _, err := storage.AddFileToIPFS(ctx, node, ipfsAPI, event.Name)
					if err != nil {
						log.Printf("Could not add file to IPFS: %s", err)
						continue
					}
					fmt.Printf("The cid of the file is %v", cidFile.String())
					// cidChan <- cidFile       // Envoyer le CID via le canal
					// fileNameChan <- fileName // Envoyer le nom du fichier via le canal
					if err := os.Remove(event.Name); err != nil { // Supprime le fichier de la Queue
						log.Print("Could not remove the file from the directory Storage_Queue/")
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					fmt.Println("error")
				}
				log.Println("EError:", err)
			}
		}
	}()

	// Watch a specific folder for changes.
	err = watcher.Add(watchDir)
	if err != nil {
		log.Println(err)
	}

	cid := <-cidChan // Recevoir le CID
	fileName := <-fileNameChan
	return cid, fileName, nil
}
