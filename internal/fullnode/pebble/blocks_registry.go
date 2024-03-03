package pebble

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/ipfs/boxo/path"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// BlockData represents the data associated with a block stored with IPFS.
type BlockData struct {
	Key []byte `json:"key"`
	Cid string `json:"cid"`
}

// BlockRegistry represents a collection of block records.
type BlockRegistry struct {
	Blocks []BlockData `json:"blocks"`
}

// saveToJSON saves the block registry records to a JSON file.
func saveToJSON(config *config.Config, filePath string, registry BlockRegistry) error {
	jsonData, err := json.Marshal(registry)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Clean(filePath), jsonData, os.FileMode(config.FileRights))
}

// loadFromJSON loads the metadata registry records from a JSON file.
func loadFromJSON(filePath string) (BlockRegistry, error) {
	var registry BlockRegistry

	jsonData, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return registry, err
	}

	err = json.Unmarshal(jsonData, &registry)
	return registry, err
}

func AddBlockMetadataToRegistry(b *block.Block, config *config.Config, fileCid path.ImmutablePath, filePath string) error {
	var metadataRegistry = BlockRegistry{}

	fileMetadata := BlockData{
		Key: block.ComputeHash(b),
		Cid: fileCid.String(),
	}

	if _, err := os.Stat(config.FileMetadataRegistryJSON); os.IsNotExist(err) {
		metadataRegistry.Blocks = append(metadataRegistry.Blocks, fileMetadata)

		if err := saveToJSON(config, config.FileMetadataRegistryJSON, metadataRegistry); err != nil {
			log.Printf("Error saving JSON data %v", err)
			return err
		}
	}

	metadataRegistry, err := loadFromJSON(config.FileMetadataRegistryJSON)
	if err != nil {
		log.Printf("Error loading JSON data %v", err)
		return err
	}

	metadataRegistry.Blocks = append(metadataRegistry.Blocks, fileMetadata)

	if err := saveToJSON(config, config.FileMetadataRegistryJSON, metadataRegistry); err != nil {
		log.Printf("Error saving JSON data %v", err)
		return err
	}

	return nil
}

func RemoveFileMetadataFromRegistry(config *config.Config, fileCid path.ImmutablePath) error {
	metadataRegistry, err := loadFromJSON(config.FileMetadataRegistryJSON)
	if err != nil {
		log.Printf("Error loading JSON data %v", err)
		return err
	}

	// Find and delete the metadata
	for i, file := range metadataRegistry.Blocks {
		if file.Cid == fileCid.String() {
			metadataRegistry.Blocks = append(metadataRegistry.Blocks[:i], metadataRegistry.Blocks[i+1:]...)
			break
		}
	}

	// Save the new metadata
	if err := saveToJSON(config, config.FileMetadataRegistryJSON, metadataRegistry); err != nil {
		log.Fatalf("Error saving JSON data: %v", err)
		return err
	}

	return nil
}
