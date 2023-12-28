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
	cfg "github.com/pierreleocadie/SecuraChain/internal/config"
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

	// Inject the plugins in the application, allowing to activate or to put plugin in the system application
	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

// Create a repo ipfs-shell/ in the home directory
func createRepo() (string, error) {
	var key = 2048

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %s", err)
	}

	ipfsPath := filepath.Join(homeDir, ".ipfs-shell")

	err = os.Mkdir(ipfsPath, cfg.FileRights)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(io.Discard, key) // Initialize a new IPFS configuration with a key of 2048 bits
	if err != nil {
		return "", err
	}

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
