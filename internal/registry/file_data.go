package registry

import (
	"os"

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
	IndexingFiles []OwnersFiles
}

// AddFileToRegistry adds a file and the data associated to the registry.
func AddFileToRegistry(log *ipfsLog.ZapEventLogger, config *config.Config, clientAnnouncement *transaction.AddFileTransaction) error {
	var indexingRegistry IndexingRegistry

	// Load existing registry if it exists
	indexingRegistry, err := LoadRegistryFile[IndexingRegistry](log, config.IndexingRegistryPath)
	if err != nil && !os.IsNotExist(err) {
		log.Errorln("Error loading indexing registry:", err)
		return err
	}

	newData := FileRegistry{
		Filename:             clientAnnouncement.Filename,
		Extension:            clientAnnouncement.Extension,
		FileSize:             clientAnnouncement.FileSize,
		Checksum:             clientAnnouncement.Checksum,
		FileCid:              clientAnnouncement.FileCid,
		TransactionTimestamp: clientAnnouncement.AnnouncementTimestamp,
	}
	log.Debugln("New FileRegistry : ", newData)

	// Check if the owner exists in the registry
	existing := false
	for i, ownerData := range indexingRegistry.IndexingFiles {
		if string(ownerData.OwnerAddress) == string(clientAnnouncement.OwnerAddress) {
			indexingRegistry.IndexingFiles[i].Files = append(ownerData.Files, newData)
			existing = true
			log.Debug("Owner exists")
			break
		}
	}

	if !existing {
		ownersData := OwnersFiles{
			OwnerAddress: clientAnnouncement.OwnerAddress,
			Files:        []FileRegistry{newData},
		}
		indexingRegistry.IndexingFiles = append(indexingRegistry.IndexingFiles, ownersData)
	}

	// Save updated registry back to file
	log.Infoln("Indexing registry created or updated successfully")
	return SaveRegistryToFile(log, config, config.IndexingRegistryPath, indexingRegistry)
}

// // saveRegistryToFile saves the indexing registry records to a JSON file.
// func saveRegistryToFile(log *ipfsLog.ZapEventLogger, config *config.Config, filePath string, registry IndexingRegistry) error {
// 	data, err := SerializeIndexingRegistry(log, registry)
// 	if err != nil {
// 		log.Errorln("Error serializing registry")
// 		return err
// 	}
// 	log.Debugln("Registry serialized successfully")
// 	return os.WriteFile(filepath.Clean(filePath), data, os.FileMode(config.FileRights))
// }

// // LoadIndexingRegistry loads the IndexingRegistry from a JSON file.
// func LoadIndexingRegistry(log *ipfsLog.ZapEventLogger, filePath string) (IndexingRegistry, error) {
// 	var registry IndexingRegistry
// 	data, err := os.ReadFile(filepath.Clean(filePath))
// 	if err != nil {
// 		log.Errorln("Error reading file %v\n", err)
// 		return registry, err
// 	}
// 	log.Debugln("Registry loaded successfully")

// 	if err := json.Unmarshal(data, &registry); err != nil {
// 		log.Errorln("Error deserializing registry")
// 		return registry, err
// 	}
// 	registry, err = DeserializeIndexingRegistry(log, data)
// 	if err != nil {
// 		log.Errorln("Error deserializing registry")
// 		return registry, err
// 	}

// 	log.Debugln("Registry deserialized successfully")
// 	return registry, err
// }

// // DeserializeIndexingRegistry converts a byte slice to a IndexingRegistry struct.
// func DeserializeIndexingRegistry(log *ipfsLog.ZapEventLogger, data []byte) (IndexingRegistry, error) {
// 	var registry IndexingRegistry
// 	if err := json.Unmarshal(data, &registry); err != nil {
// 		log.Errorln("Error deserializing registry")
// 		return registry, err
// 	}

// 	log.Debugln("Registry deserialized successfully")
// 	return registry, nil
// }

// // SerializeIndexingRegistry serializes the given IndexingRegistry into a byte slice using JSON encoding.
// func SerializeIndexingRegistry(log *ipfsLog.ZapEventLogger, registry IndexingRegistry) ([]byte, error) {
// 	log.Debugln("Serializing registry")
// 	return json.Marshal(registry)
// }

// // DeserializeFileRegistry converts a byte slice to a FileRegistry struct.
// func DeserializeFileRegistry(log *ipfsLog.ZapEventLogger, data []byte) (FileRegistry, error) {
// 	var fileRegistry FileRegistry
// 	if err := json.Unmarshal(data, &fileRegistry); err != nil {
// 		log.Errorln("Error deserializing FileRegistry")
// 		return fileRegistry, err
// 	}

// 	log.Debugln("FileRegistry deserialized successfully")
// 	return fileRegistry, nil
// }

// // SerializeFileRegistry serializes the given FileRegistry into a byte slice using JSON encoding.
// func SerializeFileRegistry(log *ipfsLog.ZapEventLogger, registry []FileRegistry) ([]byte, error) {
// 	log.Debugln("Serializing FileRegistry")
// 	return json.Marshal(registry)
// }
