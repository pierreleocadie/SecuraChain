package blockregistry

import (
	"os"

	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

type DefaultBlockRegistry struct {
	registryManager BlockRegistryManager
	Blocks          []BlockData `json:"blocks"`
	log             *ipfsLog.ZapEventLogger
	config          *config.Config
}

func NewDefaultBlockRegistry(log *ipfsLog.ZapEventLogger, config *config.Config, registryManager BlockRegistryManager) (*DefaultBlockRegistry, error) {
	blockRegistery, err := registryManager.Load(&DefaultBlockRegistry{})
	if err != nil && !os.IsNotExist(err) { // Registry does not exist/failed to load
		log.Errorln("Error loading block registry:", err)
		return nil, err
	}

	if err == nil { // Registry already exists
		log.Debugln("Block registry loaded successfully")
		blockRegistery, ok := blockRegistery.(*DefaultBlockRegistry)
		if !ok {
			log.Errorln("Error casting block registry")
			return nil, err
		}
		blockRegistery.registryManager = registryManager
		blockRegistery.log = log
		blockRegistery.config = config
		return blockRegistery, nil
	}

	return &DefaultBlockRegistry{
		registryManager: registryManager,
		log:             log,
		config:          config,
	}, nil
}

// AddBlockToRegistry adds a block and the data associated to the registry.
func (r *DefaultBlockRegistry) Add(b block.Block, fileCid path.ImmutablePath, provider peer.AddrInfo) error {
	newData := NewBlockData(b, fileCid, provider)
	r.log.Debugln("New BlockData : ", newData)

	r.Blocks = append(r.Blocks, newData)

	// Save updated registry back to file
	r.log.Infoln("Block registry created or updated successfully")
	return r.registryManager.Save(r)
}
