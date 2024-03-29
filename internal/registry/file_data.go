package registry

import (
	"encoding/base64"
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

// IndexingRegistry represents the registry for indexing files owned by a specific address.
type IndexingRegistry struct {
	IndexingFiles map[string][]FileRegistry
}

// AddFileToRegistry adds a file and the data associated to the registry.
func AddFileToRegistry(log *ipfsLog.ZapEventLogger, config *config.Config, addFileTransac *transaction.AddFileTransaction) error {
	var r IndexingRegistry

	// Load existing registry if it exists
	r, err := LoadRegistryFile[IndexingRegistry](log, config.IndexingRegistryPath)
	if err != nil && !os.IsNotExist(err) {
		log.Errorln("Error loading indexing registry:", err)
		return err
	}

	// Initialize registry if it doesn't exist
	if r.IndexingFiles == nil {
		r.IndexingFiles = make(map[string][]FileRegistry)
	}

	ownerAddressStr := base64.StdEncoding.EncodeToString(addFileTransac.OwnerAddress)

	newData := FileRegistry{
		Filename:             addFileTransac.Filename,
		Extension:            addFileTransac.Extension,
		FileSize:             addFileTransac.FileSize,
		Checksum:             addFileTransac.Checksum,
		FileCid:              addFileTransac.FileCid,
		TransactionTimestamp: addFileTransac.AnnouncementTimestamp,
	}
	log.Debugln("New FileRegistry : ", newData)

	// Ajouter le nouveau fichier à la liste existante pour cet utilisateur
	r.IndexingFiles[ownerAddressStr] = append(r.IndexingFiles[ownerAddressStr], newData)

	// Sauvegarder le registre mis à jour
	return SaveRegistryToFile(log, config, config.IndexingRegistryPath, r)
}
