package internal

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	icore "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/boxo/path"
)

func InitDetectFiles() error {
	outputBasePath := "./AddNewFiles/"
	// S'assurez que le dossier de sortie existe
	if err := os.MkdirAll(outputBasePath, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}
	fmt.Println("Dossier pour ajouter des fichiers au noeuds de stockage crée")
	return nil

}

func MonitorinRepoInit(ctx context.Context, ipfsApi icore.CoreAPI) (path.ImmutablePath, error) {
	InitDetectFiles()
	watchDir := "./AddNewFiles"

	// Create a new fsnotify watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	cidChan := make(chan path.ImmutablePath) // Canal pour passer le CID

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
					cidFile, err := AddFileToIPFS(ctx, ipfsApi, event.Name)
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
