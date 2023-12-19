package ipfs

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	icore "github.com/ipfs/boxo/coreiface"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
)

var loadPluginsOnce sync.Once

// Prepare and set up the plugins
func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins")) // Charger les plugins depuis externalPluginsPath/plugins
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	// Injecte les plugins dans l'application, permettant d'activer ou d'intégrer des plugins dans le système de l'application
	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

// Créer un répertoire IPFS
func createRepo() (string, error) {
	// Récupérer le répertoire personnel de l'utilisateur
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %s", err)
	}

	ipfsPath := filepath.Join(homeDir, ".ipfs-shell")

	err = os.Mkdir(ipfsPath, 0750) // read and execute for everyone and write for the owner
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(io.Discard, 2048) // Initialise une nouvelle configuration IPFS avec une clé de 2048 bits, io.Discard --> ignorer tous les outputs donc aucune sortie n'est affichée.
	if err != nil {
		return "", err
	}

	// // When creating the repository, you can define custom settings on the repository, such as enabling experimental
	// // features (See experimental-features.md) or customizing the gateway endpoint.
	// // To do such things, you should modify the variable `cfg`. For example:
	// // Si le drapeau expérimental est activé, elle active des fonctionnalités d'IPFS// lire les commentaires
	// //cfg permet de modifier la configuration avant de créer le dépot
	// if *flagExp {
	// 	// https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md#ipfs-filestore
	// 	cfg.Experimental.FilestoreEnabled = true
	// 	// https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md#ipfs-urlstore
	// 	cfg.Experimental.UrlstoreEnabled = true
	// 	// https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md#ipfs-p2p
	// 	cfg.Experimental.Libp2pStreamMounting = true
	// 	// https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md#p2p-http-proxy
	// 	cfg.Experimental.P2pHttpProxy = true
	// 	// See also: https://github.com/ipfs/kubo/blob/master/docs/config.md
	// 	// And: https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md
	// }

	// Create the repo with the config
	err = fsrepo.Init(ipfsPath, cfg) // Initialise le repo IPFS à l'emplacement spécifié
	if err != nil {
		return "", fmt.Errorf("failed to init node: %s", err)
	}

	return ipfsPath, nil
}

// Construct the IPFS node instance itself
func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	// Un contexte 'Context' de Go, utilisé pour la gestion de la durée de vie et l'annulation des processus
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node
	nodeOptions := &core.BuildCfg{
		Online:  true,             // Booléen indiquant si le noeud IPFS doit fonctionner en mode en ligne, ce qui signifera qu'il se connectera à d'autres noeuds IPFS sur le réseau
		Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	return core.NewNode(ctx, nodeOptions) // Crée et initialise un nouveau noeud IPFS en utilisant le context et les options spécifiées.
	// Le noeud IPFS renvoyé par cette fonction peut-être utilisé pour interargir avec le réseau IPFS, réaliser des opérations de stockage et de récupération de données, participer au DHT ETC
}

// Spawns a node
func SpawnNode(ctx context.Context) (icore.CoreAPI, *core.IpfsNode, error) {
	// Charge les plugins une seule fois
	var onceErr error
	loadPluginsOnce.Do(func() {
		onceErr = setupPlugins("")
	})
	if onceErr != nil {
		return nil, nil, onceErr
	}

	// Create a Repo IPFS
	repoPath, err := createRepo()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create the repo ./ipfs-shell: %s", err)
	}

	// Configuration et Initialisation du Noeud IPFS avec les options définies
	node, err := createNode(ctx, repoPath)
	if err != nil {
		fmt.Printf("create node failed %v", err)
		return nil, nil, err
	}

	// Créer une instance de l'API Core IPFS pour interagir avec le réseau IPFS
	api, err := coreapi.NewCoreAPI(node)

	return api, node, err
}
