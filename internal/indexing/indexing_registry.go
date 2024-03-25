package indexing

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ipfs/go-cid"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
)

// FileRegistry represents a registry of owners' files.
type FileRegistry struct {
	Filename             []byte  `json:"filename"`
	Extension            []byte  `json:"extension"`
	FileSize             uint64  `json:"fileSize"`
	Checksum             []byte  `json:"checksum"`
	FileCid              cid.Cid `json:"fileCid"`
	TransactionTimestamp int64   `json:"transactionTimestamp"`
}

// OwnersFiles represents a collection of files owned by a specific address.
type OwnersFiles struct {
	OwnerAddress []byte
	Files        []FileRegistry
}

// IndexingRegistry represents the registry for indexing files owned by a specific address.
type IndexingRegistry struct {
	OwnersFilesData []OwnersFiles
}

// AddFileToRegistry adds a file and the data associated to the registry.
func AddFileToRegistry(log *ipfsLog.ZapEventLogger, config *config.Config, clientAnnouncement *transaction.ClientAnnouncement) error {
	registry := IndexingRegistry{}

	newData := FileRegistry{
		Filename:             clientAnnouncement.Filename,
		Extension:            clientAnnouncement.Extension,
		FileSize:             clientAnnouncement.FileSize,
		Checksum:             clientAnnouncement.Checksum,
		FileCid:              clientAnnouncement.FileCid,
		TransactionTimestamp: clientAnnouncement.AnnouncementTimestamp,
	}

	ownersData := OwnersFiles{
		OwnerAddress: clientAnnouncement.OwnerAddress,
		Files:        []FileRegistry{newData},
	}

	log.Debugln("New FileRegistry : ", newData)

	if _, err := os.Stat(config.IndexingRegistryPath); os.IsNotExist(err) {
		registry.OwnersFilesData = append(registry.OwnersFilesData, ownersData)

		if err := saveRegistryToFile(log, config, config.IndexingRegistryPath, registry); err != nil {
			log.Errorln("Error saving JSON data %v", err)
			return err
		}

		log.Infoln("Block registry created successfully")
		return nil
	}

	indexingRegistry, err := LoadIndexingRegistry(log, config.IndexingRegistryPath)
	if err != nil {
		log.Errorln("Error loading JSON data %v", err)
		return err
	}

	indexingRegistry.OwnersFilesData = append(indexingRegistry.OwnersFilesData, ownersData)

	if err := saveRegistryToFile(log, config, config.IndexingRegistryPath, indexingRegistry); err != nil {
		log.Errorln("Error saving JSON data %v", err)
		return err
	}

	log.Infoln("Block registry updated successfully")
	return nil
}

// saveRegistryToFile saves the block registry records to a JSON file.
func saveRegistryToFile(log *ipfsLog.ZapEventLogger, config *config.Config, filePath string, registry IndexingRegistry) error {
	data, err := SerializeIndexingRegistry(log, registry)
	if err != nil {
		log.Errorln("Error serializing registry")
		return err
	}
	log.Debugln("Registry serialized successfully")
	return os.WriteFile(filepath.Clean(filePath), data, os.FileMode(config.FileRights))
}

// LoadIndexingRegistry loads the IndexingRegistry from a JSON file.
func LoadIndexingRegistry(log *ipfsLog.ZapEventLogger, filePath string) (IndexingRegistry, error) {
	var registry IndexingRegistry
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
	registry, err = DeserializeIndexingRegistry(log, data)
	if err != nil {
		log.Errorln("Error deserializing registry")
		return registry, err
	}

	log.Debugln("Registry deserialized successfully")
	return registry, err
}

// DeserializeIndexingRegistry converts a byte slice to a IndexingRegistry struct.
func DeserializeIndexingRegistry(log *ipfsLog.ZapEventLogger, data []byte) (IndexingRegistry, error) {
	var registry IndexingRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		log.Errorln("Error deserializing registry")
		return registry, err
	}

	log.Debugln("Registry deserialized successfully")
	return registry, nil
}

// SerializeIndexingRegistry serializes the given IndexingRegistry into a byte slice using JSON encoding.
func SerializeIndexingRegistry(log *ipfsLog.ZapEventLogger, registry IndexingRegistry) ([]byte, error) {
	log.Debugln("Serializing registry")
	return json.Marshal(registry)
}
