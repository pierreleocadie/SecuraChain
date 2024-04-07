package ipfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	inconfig "github.com/pierreleocadie/SecuraChain/internal/config"
)

var loadPluginsOnce sync.Once

// Prepare and set up the plugins
func setupPlugins(log *ipfsLog.ZapEventLogger, externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins")) // Charger les plugins depuis externalPluginsPath/plugins
	if err != nil {
		log.Errorln("error loading plugins")
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		log.Errorln("error initializing plugins")
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	// Inject the plugins in the application, allowing to activate or to put plugin in the system application
	if err := plugins.Inject(); err != nil {
		log.Errorln("error initializing plugins")
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	log.Debugln("Plugins loaded and injected")
	return nil
}

// Create a repo ipfs-shell/ in the home directory
func createRepo(log *ipfsLog.ZapEventLogger, cfgg *inconfig.Config) (string, error) {
	var key = 2048

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Errorln("failed to get user home directory")
		return "", fmt.Errorf("failed to get user home directory: %s", err)
	}

	ipfsPath := filepath.Join(homeDir, ".ipfs-shell")

	err = os.Mkdir(ipfsPath, os.FileMode(cfgg.FileRights))
	if err != nil && !os.IsExist(err) {
		log.Errorln("failed to create ipfs-shell directory")
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(io.Discard, key) // Initialize a new IPFS configuration with a key of 2048 bits
	if err != nil {
		log.Errorln("failed to init config")
		return "", err
	}

	// Create the repo with the config
	err = fsrepo.Init(ipfsPath, cfg) // Initialize the directory with the config given
	if err != nil {
		log.Errorln("failed to init node")
		return "", fmt.Errorf("failed to init node: %s", err)
	}

	log.Debugln("Repo created at", ipfsPath)
	return ipfsPath, nil
}

// Construct the IPFS node instance itself
func createNode(log *ipfsLog.ZapEventLogger, ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		log.Errorln("failed to open repo")
		return nil, err
	}

	// Construct the node
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption,
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	log.Debugln("Creating node")
	return core.NewNode(ctx, nodeOptions) // Create and initialize a new node IPFS
}

// Spawns a node
func SpawnNode(log *ipfsLog.ZapEventLogger, ctx context.Context, cfgg *inconfig.Config) (icore.CoreAPI, *core.IpfsNode, error) {
	// Load the plugins once
	var onceErr error

	loadPluginsOnce.Do(func() {
		onceErr = setupPlugins(log, "")
	})

	if onceErr != nil {
		log.Errorln("failed to load plugins")
		return nil, nil, onceErr
	}

	// Create a Repo IPFS
	repoPath, err := createRepo(log, cfgg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create the repo ./ipfs-shell: %s", err)
	}

	// Initialization and Configuration of a node IPFS
	node, err := createNode(log, ctx, repoPath)
	if err != nil {
		log.Errorf("create node failed %v\n", err)
		return nil, nil, err
	}

	// Create an instance of the Core IPFS API to interact with the IPFS network
	coreAPI, err := coreapi.NewCoreAPI(node)
	if err != nil {
		log.Errorln("failed to create core API")
		return nil, nil, err
	}

	log.Debugln("Node created and Core API instance created")
	return coreAPI, node, err
}
