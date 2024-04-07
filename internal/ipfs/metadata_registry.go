package ipfs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
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
func saveToJSON(log *ipfsLog.ZapEventLogger, config *config.Config, filePath string, registry MetadataRegistry) error {
	jsonData, err := json.Marshal(registry)
	if err != nil {
		log.Errorln("Error marshalling JSON data %v", err)
		return err
	}

	log.Debugln("Saving metadata registry to", filePath)
	return os.WriteFile(filepath.Clean(filePath), jsonData, os.FileMode(config.FileRights))
}

// loadFromJSON loads the metadata registry records from a JSON file.
func loadFromJSON(log *ipfsLog.ZapEventLogger, filePath string) (MetadataRegistry, error) {
	var registry MetadataRegistry

	jsonData, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		log.Errorln("Error reading JSON data %v", err)
		return registry, err
	}

	err = json.Unmarshal(jsonData, &registry)
	log.Debugln("Loading metadata registry from", filePath)
	return registry, err
}

func AddFileMetadataToRegistry(log *ipfsLog.ZapEventLogger, config *config.Config, fileCid path.ImmutablePath, filePath string) error {
	var metadataRegistry = MetadataRegistry{}

	fileName, fileSize, fileType, err := utils.FileInfo(log, filePath)
	if err != nil {
		log.Errorln("Error getting file info: %v", err)
	}

	fileMetadata := FileMetadata{
		Cid:       fileCid.String(),
		Timestamp: time.Now(),
		Size:      fileSize,
		Extension: fileType,
		Name:      fileName,
	}

	if _, err := os.Stat(config.FileMetadataRegistryJSON); os.IsNotExist(err) {
		metadataRegistry.Files = append(metadataRegistry.Files, fileMetadata)

		if err := saveToJSON(log, config, config.FileMetadataRegistryJSON, metadataRegistry); err != nil {
			log.Errorln("Error saving JSON data %v", err)
			return err
		}
	}

	metadataRegistry, err = loadFromJSON(log, config.FileMetadataRegistryJSON)
	if err != nil {
		log.Errorln("Error loading JSON data %v", err)
		return err
	}

	metadataRegistry.Files = append(metadataRegistry.Files, fileMetadata)

	if err := saveToJSON(log, config, config.FileMetadataRegistryJSON, metadataRegistry); err != nil {
		log.Errorln("Error saving JSON data %v", err)
		return err
	}

	log.Debugln("File metadata added to registry")
	return nil
}

func RemoveFileMetadataFromRegistry(log *ipfsLog.ZapEventLogger, config *config.Config, fileCid path.ImmutablePath) error {
	metadataRegistry, err := loadFromJSON(log, config.FileMetadataRegistryJSON)
	if err != nil {
		log.Errorln("Error loading JSON data %v", err)
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
	if err := saveToJSON(log, config, config.FileMetadataRegistryJSON, metadataRegistry); err != nil {
		log.Errorln("Error saving JSON data: %v", err)
		return err
	}

	log.Debugln("File metadata removed from registry")
	return nil
}
