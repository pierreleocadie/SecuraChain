package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	icore "github.com/ipfs/boxo/coreiface"
	files "github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	// Assurez-vous d'avoir importé le bon package pour path
)

// Pour récupérer des fichiers ou des dossiers IPFS en utilisant leur CID
func FetchFileFromIPFS(ctx context.Context, ipfsApi icore.CoreAPI, cidFile path.ImmutablePath) error {
	outputBasePath := "./Ipfs-files-downloaded/"
	// S'assurez que le dossier de sortie existe
	if err := os.MkdirAll(outputBasePath, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	rootNodeFile, err := ipfsApi.Unixfs().Get(ctx, cidFile)
	if err != nil {
		return fmt.Errorf("could not get file with CID: %s", err)
	}

	outputPath := filepath.Join(outputBasePath, filepath.Base(cidFile.String()))

	err = files.WriteTo(rootNodeFile, outputPath)
	if err != nil {
		return fmt.Errorf("could not write out the fetched CID: %v", err)
	}

	fmt.Printf("Got file back from IPFS (IPFS path: %s) and wrote it to %s\n", cidFile.String(), outputPath)

	return nil
}

//Pour récupérer des répertoires IPFS en utilisant leur CID

func FetchDirectoryFromIPFS(ctx context.Context, ipfsApi icore.CoreAPI, cidDirectory path.ImmutablePath) error {
	outputBasePathDir := "./Ipfs-files-downloaded/Dir/"
	// S'assurez que le dossier de sortie existe
	if err := os.MkdirAll(outputBasePathDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	rootNodeDirectory, err := ipfsApi.Unixfs().Get(ctx, cidDirectory)
	if err != nil {
		return fmt.Errorf("could not get file with CID: %s", err)
	}

	outputPathDirectory := filepath.Join(outputBasePathDir, filepath.Base(cidDirectory.String()))

	err = files.WriteTo(rootNodeDirectory, outputPathDirectory)
	if err != nil {
		return fmt.Errorf("could not write out the fetched CID: %v", err)
	}

	fmt.Printf("Got directory back from IPFS (IPFS path: %s) and wrote it to %s\n", cidDirectory.String(), outputPathDirectory)

	return nil
}
