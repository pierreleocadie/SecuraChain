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
	outputBasePath := "./Ipfs-files-downloaded/files"
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

func FetchFileFromIPFSNetwork(ctx context.Context, ipfsApi icore.CoreAPI, peerCidFile path.ImmutablePath) error {
	outputBasePathNetwork := "./Ipfs-files-downloaded/Network/"

	// S'assurez que le dossier de sortie existe
	if err := os.MkdirAll(outputBasePathNetwork, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}
	exampleCIDStr := peerCidFile.RootCid().String()

	fmt.Printf("Fetching a file from the network with CID %s\n", exampleCIDStr)
	outputPath := outputBasePathNetwork + exampleCIDStr
	testCID := path.FromCid(peerCidFile.RootCid())

	rootNode, err := ipfsApi.Unixfs().Get(ctx, testCID)
	if err != nil {
		return fmt.Errorf("could not get file with CID: %s", err)
	}

	err = files.WriteTo(rootNode, outputPath)
	if err != nil {
		panic(fmt.Errorf("could not write out the fetched CID: %s", err))
	}

	fmt.Printf("Wrote the file to %s\n", outputPath)

	return nil
}
