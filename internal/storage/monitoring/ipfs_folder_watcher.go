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

// CreateStorageQueueDirectory creates a directory for adding files to the storage node.
func CreateStorageQueueDirectory() error {
	outputBasePath := "./Storage_Queue"
	// S'assurez que le dossier de sortie existe
	if err := os.MkdirAll(outputBasePath, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}
	fmt.Println("Dossier pour ajouter des fichiers au noeuds de stockage crée")
	return nil

}

// MonitorinRepoInit watches a directory for new files and adds them to IPFS.
// It initializes the directory for monitoring, creates a watcher, and processes file events.
func WatchStorageQueueForChanges(ctx context.Context, node *core.IpfsNode, ipfsApi icore.CoreAPI) (path.ImmutablePath, error) {
	CreateStorageQueueDirectory()
	watchDir := "./Storage_Queue"

	// Create a new fsnotify watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	cidChan := make(chan path.ImmutablePath) // Channel for passing the CID

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Check the type of event and log the details.
				fmt.Printf("event: %v\n", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Printf("modified file: %s\n", event.Name)
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					fmt.Printf("created file: %s\n", event.Name)
					cidFile, err := storage.AddFileToIPFS(ctx, node, ipfsApi, event.Name)
					if err != nil {
						log.Printf("Could not add file to IPFS: %s", err)
						continue
					}
					fmt.Printf("The cid of the file is %v", cidFile.String())
					cidChan <- cidFile // Envoyer le CID via le canal
					return

				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					fmt.Printf("deleted file: %s\n", event.Name)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	// Watch a specific folder for changes.
	err = watcher.Add(watchDir)
	if err != nil {
		log.Fatal(err)
	}

	cid := <-cidChan // Recevoir le CID
	return cid, nil
}
