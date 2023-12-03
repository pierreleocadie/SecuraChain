package internal

import (
	"context"
	"fmt"
	"os"

	icore "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
)

// Prepare the file to be added to IPFS
func getUnixfsNode(path string) (files.Node, error) {
	st, err := os.Stat(path) // recupère les informations sur le fichier ou le répertoire situé au chemin 'path', retourne une structure 'FileInfo' avec des détails sur la taille du fichier , les permissions etc...
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st) // créer un noeud de fichier UnixFS à partir du chemin donné, 1re par accès au fichier, 2em si le fichier doit être traté comme un fichier caché si oui alors 'false', 3eme structure 'FileInfo' obtenu avant
	if err != nil {
		return nil, err
	}

	return f, nil
}

// Pour ajouter des fichiers à IPFS
// inputPathFile --> where the file is
func AddFileToIPFS(ctx context.Context, ipfsApi icore.CoreAPI, inputPathFile string) (path.ImmutablePath, error) {
	someFile, err := getUnixfsNode(inputPathFile)
	if err != nil {
		panic(fmt.Errorf("could not get File: %s", err))
	}

	cidFile, err := ipfsApi.Unixfs().Add(ctx, someFile)
	if err != nil {
		panic(fmt.Errorf("could not add File: %s", err))
	}
	fmt.Printf("Added file to IPFS with CID %s\n", cidFile.String())

	return cidFile, nil
}
