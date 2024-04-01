package registry

import (
	"bytes"
	"fmt"
	"os"

	"github.com/ipfs/go-cid"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
)

// FileRegistry represents a registry of owners' files.
type FileRegistry struct {
	Filename             []byte          `json:"filename"`
	Extension            []byte          `json:"extension"`
	FileSize             uint64          `json:"fileSize"`
	Checksum             []byte          `json:"checksum"`
	Providers            []peer.AddrInfo `json:"providers"`
	FileCid              cid.Cid         `json:"fileCid"`
	TransactionTimestamp int64           `json:"transactionTimestamp"`
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

	ownerAddressStr := fmt.Sprintf("%x", addFileTransac.OwnerAddress)
	log.Debugln("Owner address converted to string : ", ownerAddressStr)

	// If the file already exists, add the new provider to the list of providers
	for i, file := range r.IndexingFiles[ownerAddressStr] {
		if bytes.Equal(file.Filename, addFileTransac.Filename) && bytes.Equal(file.Extension, addFileTransac.Extension) && file.FileSize == addFileTransac.FileSize && bytes.Equal(file.Checksum, addFileTransac.Checksum) && file.FileCid == addFileTransac.FileCid {
			r.IndexingFiles[ownerAddressStr][i].Providers = append(r.IndexingFiles[ownerAddressStr][i].Providers, addFileTransac.IPFSStorageNodeAddrInfo)
			log.Debugln("[AddFileToRegistry] - File already exists, new provider added to the list of providers")
			return SaveRegistryToFile(log, config, config.IndexingRegistryPath, r)
		}
	}

	newData := FileRegistry{
		Filename:             addFileTransac.Filename,
		Extension:            addFileTransac.Extension,
		FileSize:             addFileTransac.FileSize,
		Checksum:             addFileTransac.Checksum,
		Providers:            []peer.AddrInfo{addFileTransac.IPFSStorageNodeAddrInfo},
		FileCid:              addFileTransac.FileCid,
		TransactionTimestamp: addFileTransac.AnnouncementTimestamp,
	}
	log.Debugln("New FileRegistry : ", newData)

	// Add the file to the list for this user
	r.IndexingFiles[ownerAddressStr] = append(r.IndexingFiles[ownerAddressStr], newData)

	log.Debugln("[AddFileToRegistry] - File added to registry")

	// Save the updated registry
	return SaveRegistryToFile(log, config, config.IndexingRegistryPath, r)
}

// DeleteFileFromRegistry deletes a file and the data associated from the registry.
func DeleteFileFromRegistry(log *ipfsLog.ZapEventLogger, config *config.Config, deleteFileTransac *transaction.DeleteFileTransaction) error {
	var r IndexingRegistry

	// Load existing registry if it exists
	r, err := LoadRegistryFile[IndexingRegistry](log, config.IndexingRegistryPath)
	if err != nil && !os.IsNotExist(err) {
		log.Errorln("Error loading indexing registry:", err)
		return err
	}

	ownerAddressStr := fmt.Sprintf("%x", deleteFileTransac.OwnerAddress)
	log.Debugln("Owner address converted to string : ", ownerAddressStr)

	// Delete the file from the list for this user
	for i, file := range r.IndexingFiles[ownerAddressStr] {
		if file.FileCid == deleteFileTransac.FileCid {
			r.IndexingFiles[ownerAddressStr] = append(r.IndexingFiles[ownerAddressStr][:i], r.IndexingFiles[ownerAddressStr][i+1:]...)
			break
		}
	}

	log.Debugln("[AddFileToRegistry] - File deleted from the registry")
	// Save the updated registry
	return SaveRegistryToFile(log, config, config.IndexingRegistryPath, r)
}
