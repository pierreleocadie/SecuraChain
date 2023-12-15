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

// IPFS ne permet pas la suppression traditionnelle de fichiers
// Mais on peut gérer le retraut des pins ou la gestion de cache
// On peut aussi supprimer
func DeleteFromIPFS(ctx context.Context, ipfsApi icore.CoreAPI, cid path.ImmutablePath, fileName string) error {
	// Unpin the file on IPFS
	ipfsApi.Pin().Rm(ctx, cid)
	_, IsUnPinned, err := ipfsApi.Pin().IsPinned(ctx, cid)
	if err != nil {
		return err
	}

	fmt.Println("Le fichier a bien été unpin avec le cid: ", IsUnPinned)

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
	// Trouver et supprimer la métadonnée
	for i, file := range fileData.Files {
		if file.OriginalName == fileName {
			fileData.Files = append(fileData.Files[:i], fileData.Files[i+1:]...)
			break
		}
	}

	// Sauvegarder les données mises à jour
	if err := util.SaveToJSON(fileNameJSON, fileData); err != nil {
		log.Fatalf("Error saving JSON data: %v", err)
	}

	fmt.Println("Métadonnée supprimée avec succès")
	return nil
}
