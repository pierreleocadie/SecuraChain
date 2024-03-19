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
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
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

		if err := saveRegistryToFile(log, config, config.RegistryPath, registry); err != nil {
			log.Errorln("Error saving JSON data %v", err)
			return err
		}

		log.Infoln("Block registry created successfully")
		return nil
	}

	metadataRegistry, err := LoadRegistry(log, config.RegistryPath)
	if err != nil {
		log.Errorln("Error loading JSON data %v", err)
		return err
	}

	metadataRegistry.Blocks = append(metadataRegistry.Blocks, newData)

	if err := saveRegistryToFile(log, config, config.RegistryPath, metadataRegistry); err != nil {
		log.Errorln("Error saving JSON data %v", err)
		return err
	}

	log.Infoln("Block registry updated successfully")
	return nil
}

// saveRegistryToFile saves the block registry records to a JSON file.
func saveRegistryToFile(log *ipfsLog.ZapEventLogger, config *config.Config, filePath string, registry BlockRegistry) error {
	data, err := SerializeRegistry(log, registry)
	if err != nil {
		log.Errorln("Error serializing registry")
		return err
	}
	log.Debugln("Registry serialized successfully")
	return os.WriteFile(filepath.Clean(filePath), data, os.FileMode(config.FileRights))
}

// LoadRegistry loads the block registry from a JSON file.
func LoadRegistry(log *ipfsLog.ZapEventLogger, filePath string) (BlockRegistry, error) {
	var registry BlockRegistry
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		log.Errorln("Error reading file %v\n", err)
		return registry, err
	}
	log.Debugln("Registry loaded successfully")

	if err := json.Unmarshal(data, &registry); err != nil {
		log.Errorln("Error deserializing registry")
		return registry, err
	}
	registry, err = DeserializeRegistry(log, data)
	if err != nil {
		log.Errorln("Error deserializing registry")
		return registry, err
	}

	log.Debugln("Registry deserialized successfully")
	return registry, err
}

// DeserializeRegistry converts a byte slice to a BlockRegistry struct.
func DeserializeRegistry(log *ipfsLog.ZapEventLogger, data []byte) (BlockRegistry, error) {
	var registry BlockRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		log.Errorln("Error deserializing registry")
		return registry, err
	}

	log.Debugln("Registry deserialized successfully")
	return registry, nil
}

// SerializeRegistry serializes the given BlockRegistry into a byte slice using JSON encoding.
func SerializeRegistry(log *ipfsLog.ZapEventLogger, registry BlockRegistry) ([]byte, error) {
	log.Debugln("Serializing registry")
	return json.Marshal(registry)
}

// ConvertToBlock reads the contents of the file at the given file path and converts it into a block.Block object.
func ConvertToBlock(log *ipfsLog.ZapEventLogger, filePath string) (*block.Block, error) {
	filePath, err := utils.SanitizePath(filePath)
	if err != nil {
		log.Errorf("Error sanitizing file path %v\n", err)
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Errorf("Error reading file %v\n", err)
		return nil, err
	}

	log.Debugln("File converted to block successfully")
	return block.DeserializeBlock(data)
}
