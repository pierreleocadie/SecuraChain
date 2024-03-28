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
func AddFileToRegistry(log *ipfsLog.ZapEventLogger, config *config.Config, addFileTransac *transaction.AddFileTransaction) error {
	var indexingRegistry IndexingRegistry

	// Load existing registry if it exists
	indexingRegistry, err := LoadRegistryFile[IndexingRegistry](log, config.IndexingRegistryPath)
	if err != nil && !os.IsNotExist(err) {
		log.Errorln("Error loading indexing registry:", err)
		return err
	}

	newData := FileRegistry{
		Filename:             addFileTransac.Filename,
		Extension:            addFileTransac.Extension,
		FileSize:             addFileTransac.FileSize,
		Checksum:             addFileTransac.Checksum,
		FileCid:              addFileTransac.FileCid,
		TransactionTimestamp: addFileTransac.AnnouncementTimestamp,
	}
	log.Debugln("New FileRegistry : ", newData)

	// Check if the owner exists in the registry
	existing := false
	for i, ownerData := range indexingRegistry.IndexingFiles {
		if string(ownerData.OwnerAddress) == string(addFileTransac.OwnerAddress) {
			indexingRegistry.IndexingFiles[i].Files = append(ownerData.Files, newData)
			existing = true
			log.Debug("Owner exists")
			break
		}
	}

	if !existing {
		ownersData := OwnersFiles{
			OwnerAddress: addFileTransac.OwnerAddress,
			Files:        []FileRegistry{newData},
		}
		indexingRegistry.IndexingFiles = append(indexingRegistry.IndexingFiles, ownersData)
	}

	// Save updated registry back to file
	log.Infoln("Indexing registry created or updated successfully")
	return SaveRegistryToFile(log, config, config.IndexingRegistryPath, indexingRegistry)
}
