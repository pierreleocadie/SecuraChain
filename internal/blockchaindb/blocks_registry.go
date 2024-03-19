package blockchaindb

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// BlockData represents the data associated with a block stored with IPFS.
type BlockData struct {
	ID       uint32        `json:"ID"`
	Key      []byte        `json:"key"`
	BlockCid cid.Cid       `json:"cid"`
	Provider peer.AddrInfo `json:"provider"`
}

// BlockRegistry represents a collection of block records.
type BlockRegistry struct {
	Blocks []BlockData `json:"blocks"`
}

// AddBlockToRegistry adds a block and the data associated to the registry.
func AddBlockToRegistry(log *ipfsLog.ZapEventLogger, b *block.Block, config *config.Config, fileCid path.ImmutablePath, provider peer.AddrInfo) error {
	registry := BlockRegistry{}

	newData := BlockData{
		ID:       b.Header.Height,
		Key:      block.ComputeHash(b),
		BlockCid: fileCid.RootCid(),
		Provider: provider,
	}
	log.Debugln("New BlockData : ", newData)

	if _, err := os.Stat(config.RegistryPath); os.IsNotExist(err) {
		registry.Blocks = append(registry.Blocks, newData)

		if err := saveRegistryToFile(config, config.RegistryPath, registry); err != nil {
			log.Errorln("Error saving JSON data %v", err)
			return err
		}

		log.Debugln("Block registry created successfully")
		return nil
	}

	metadataRegistry, err := LoadRegistry(config.RegistryPath)
	if err != nil {
		log.Errorln("Error loading JSON data %v", err)
		return err
	}

	metadataRegistry.Blocks = append(metadataRegistry.Blocks, newData)

	if err := saveRegistryToFile(config, config.RegistryPath, metadataRegistry); err != nil {
		log.Errorln("Error saving JSON data %v", err)
		return err
	}

	log.Debugln("Block registry updated successfully")
	return nil
}

// saveRegistryToFile saves the block registry records to a JSON file.
func saveRegistryToFile(config *config.Config, filePath string, registry BlockRegistry) error {
	data, err := json.Marshal(registry)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Clean(filePath), data, os.FileMode(config.FileRights))
}

// LoadRegistry loads the block registry from a JSON file.
func LoadRegistry(filePath string) (BlockRegistry, error) {
	var registry BlockRegistry
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return registry, err
	}

	if err := json.Unmarshal(data, &registry); err != nil {
		return registry, err
	}
	return registry, err
}

// DeserializeRegistry converts a byte slice to a BlockRegistry struct.
func DeserializeRegistry(data []byte) (BlockRegistry, error) {
	var registry BlockRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return registry, err
	}
	return registry, nil
}

// SerializeRegistry serializes the given BlockRegistry into a byte slice using JSON encoding.
func SerializeRegistry(registry BlockRegistry) ([]byte, error) {
	return json.Marshal(registry)
}

// ConvertToBlock reads the contents of the file at the given file path and converts it into a block.Block object.
func ConvertToBlock(filePath string) (*block.Block, error) {
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, err
	}
	return block.DeserializeBlock(data)
}
