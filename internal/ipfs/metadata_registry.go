package ipfs

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ipfs/boxo/path"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

// FileMetaData represents the metadata associated with a file stored in the storage node.
type FileMetadata struct {
	Cid       string    `json:"cid"`       //
	Timestamp time.Time `json:"timestamp"` // Timestamp of when the file was added.
	Size      string    `json:"size"`      // Size of the file in bytes.
	Extension string    `json:"extension"` // L'extension du fichier.
	Name      string    `json:"name"`      // Original name of the file.
}

// MetadataRegistry represents a collection of file metadata records.
type MetadataRegistry struct {
	Files []FileMetadata `json:"files"`
}

// saveToJSON saves the metadata registry records to a JSON file.
func saveToJSON(filePath string, registry MetadataRegistry) error {
	jsonData, err := json.Marshal(registry)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Clean(filePath), jsonData, config.FileRights)
}

// loadFromJSON loads the metadata registry records from a JSON file.
func loadFromJSON(filePath string) (MetadataRegistry, error) {
	var registry MetadataRegistry

	jsonData, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return registry, err
	}

	err = json.Unmarshal(jsonData, &registry)
	return registry, err
}

func AddFileMetadataToRegistry(fileCid path.ImmutablePath, filePath string) error {
	var metadataRegistry = MetadataRegistry{}

	fileName, fileSize, fileType, err := utils.FileInfo(filePath)
	if err != nil {
		log.Fatalf("Error getting file info: %v", err)
	}

	fileMetadata := FileMetadata{
		Cid:       fileCid.String(),
		Timestamp: time.Now(),
		Size:      fileSize,
		Extension: fileType,
		Name:      fileName,
	}

	if _, err := os.Stat(config.FileMetadataRegistryJson); os.IsNotExist(err) {
		metadataRegistry.Files = append(metadataRegistry.Files, fileMetadata)

		if err := saveToJSON(config.FileMetadataRegistryJson, metadataRegistry); err != nil {
			log.Printf("Error saving JSON data %v", err)
			return err
		}
	}

	metadataRegistry, err = loadFromJSON(config.FileMetadataRegistryJson)
	if err != nil {
		log.Printf("Error loading JSON data %v", err)
		return err
	}

	metadataRegistry.Files = append(metadataRegistry.Files, fileMetadata)

	if err := saveToJSON(config.FileMetadataRegistryJson, metadataRegistry); err != nil {
		log.Printf("Error saving JSON data %v", err)
		return err
	}

	return nil
}

func RemoveFileMetadataFromRegistry(fileCid path.ImmutablePath) error {
	metadataRegistry, err := loadFromJSON(config.FileMetadataRegistryJson)
	if err != nil {
		log.Printf("Error loading JSON data %v", err)
		return err
	}

	// Find and delete the metadata
	for i, file := range metadataRegistry.Files {
		if file.Cid == fileCid.String() {
			metadataRegistry.Files = append(metadataRegistry.Files[:i], metadataRegistry.Files[i+1:]...)
			break
		}
	}

	// Save the new metadata
	if err := saveToJSON(config.FileMetadataRegistryJson, metadataRegistry); err != nil {
		log.Fatalf("Error saving JSON data: %v", err)
		return err
	}

	return nil
}
