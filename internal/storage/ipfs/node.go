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

const permission = 0700 // Only the owner can read, write and execute

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

	// Inject the plugins in the application, allowing to activate or to put plugin in the system application
	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

// Create a repo ipfs-shell/ in the home directory
func createRepo() (string, error) {
	var key = 2048

	// take the home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %s", err)
	}

	ipfsPath := filepath.Join(homeDir, ".ipfs-shell")

	err = os.Mkdir(ipfsPath, permission) // read and execute for everyone and write for the owner
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(io.Discard, key) // Initialize a new IPFS configuration with a key of 2048 bits
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
	err = fsrepo.Init(ipfsPath, cfg) // Initialize the directory with the config given
	if err != nil {
		return "", fmt.Errorf("failed to init node: %s", err)
	}

	return ipfsPath, nil
}

// Construct the IPFS node instance itself
func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption,
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	return core.NewNode(ctx, nodeOptions) // Create and initialize a new node IPFS
}

// Spawns a node
func SpawnNode(ctx context.Context) (icore.CoreAPI, *core.IpfsNode, error) {
	// Load the plugins once
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

	// Initialization and Configuration of a node IPFS
	node, err := createNode(ctx, repoPath)
	if err != nil {
		fmt.Printf("create node failed %v", err)
		return nil, nil, err
	}

	// Create an instance of the Core IPFS API to interact with the IPFS network
	api, err := coreapi.NewCoreAPI(node)

	return api, node, err
}
