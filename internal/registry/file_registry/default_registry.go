package fileregistry

import (
	"bytes"
	"fmt"
	"os"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
)

// FileRegistry represents the registry for indexing files owned by a specific address.
type DefaultFileRegistry struct {
	registryManager FileRegistryManager
	IndexingFiles   map[string][]FileData
	log             *ipfsLog.ZapEventLogger
	config          *config.Config
}

func NewDefaultFileRegistry(log *ipfsLog.ZapEventLogger, config *config.Config, registryManager FileRegistryManager) (*DefaultFileRegistry, error) {
	fileRegistery, err := registryManager.Load(&DefaultFileRegistry{})
	if err != nil && !os.IsNotExist(err) { // Registry does not exist/failed to load
		log.Errorln("Error loading block registry:", err)
		return nil, err
	}

	if err == nil { // Registry already exists
		log.Debugln("File registry loaded successfully")
		return fileRegistery.(*DefaultFileRegistry), nil
	}

	return &DefaultFileRegistry{
		registryManager: registryManager,
		IndexingFiles:   make(map[string][]FileData),
		log:             log,
		config:          config,
	}, nil
}

// AddFileToRegistry adds a file and the data associated to the registry.
func (r *DefaultFileRegistry) Add(addFileTransac transaction.AddFileTransaction) error {
	ownerAddressStr := fmt.Sprintf("%x", addFileTransac.OwnerAddress)
	r.log.Debugln("Owner address converted to string : ", ownerAddressStr)

	for i, file := range r.IndexingFiles[ownerAddressStr] {
		if bytes.Equal(file.Filename, addFileTransac.Filename) && bytes.Equal(file.Extension, addFileTransac.Extension) && file.FileSize == addFileTransac.FileSize && bytes.Equal(file.Checksum, addFileTransac.Checksum) && file.FileCid == addFileTransac.FileCid {
			r.IndexingFiles[ownerAddressStr][i].Providers = append(r.IndexingFiles[ownerAddressStr][i].Providers, addFileTransac.IPFSStorageNodeAddrInfo)
			r.log.Debugln("[AddFileToRegistry] - File already exists, new provider added to the list of providers")
			return r.registryManager.Save(r)
		}
	}

	newData := NewFileData(addFileTransac)
	r.log.Debugln("New FileRegistry : ", newData)

	// Add the file to the list for this user
	r.IndexingFiles[ownerAddressStr] = append(r.IndexingFiles[ownerAddressStr], newData)
	r.log.Debugln("[AddFileToRegistry] - File added to registry")

	// Save the updated registry
	return r.registryManager.Save(r)
}

// DeleteFileFromRegistry deletes a file and the data associated from the registry.
func (r *DefaultFileRegistry) Delete(deleteFileTransac transaction.DeleteFileTransaction) error {
	ownerAddressStr := fmt.Sprintf("%x", deleteFileTransac.OwnerAddress)
	r.log.Debugln("Owner address converted to string : ", ownerAddressStr)

	// Delete the file from the list for this user
	for i, file := range r.IndexingFiles[ownerAddressStr] {
		if file.FileCid == deleteFileTransac.FileCid {
			r.IndexingFiles[ownerAddressStr] = append(r.IndexingFiles[ownerAddressStr][:i], r.IndexingFiles[ownerAddressStr][i+1:]...)
			break
		}
	}

	r.log.Debugln("[DeleteFileToRegistry] - File deleted from the registry")
	// Save the updated registry
	return r.registryManager.Save(r)
}

// searchMyFiles searches for files owned by a specific user (by his publicKey).
func (r DefaultFileRegistry) Get(myPublicKey string) []FileData {
	r.log.Debugln("Searching for files of owner : ", myPublicKey)
	var myFiles []FileData

	for owner, filesRegistry := range r.IndexingFiles {
		if owner == myPublicKey {
			myFiles = append(myFiles, filesRegistry...)
			r.log.Debugln("Files found")
			return myFiles
		}
	}

	r.log.Debugln("No files found")
	return myFiles
}

func (r *DefaultFileRegistry) UpdateFromBlock(b block.Block) error {
	for _, tx := range b.Transactions {
		switch tx.(type) {
		case *transaction.AddFileTransaction:
			addFileTransac := tx.(transaction.AddFileTransaction)
			if err := r.Add(addFileTransac); err != nil {
				return fmt.Errorf("Error adding file to registry: %v", err)
			}
			r.log.Debugln("File added to the registry")
		case *transaction.DeleteFileTransaction:
			deleteFileTransac := tx.(transaction.DeleteFileTransaction)
			if err := r.Delete(deleteFileTransac); err != nil {
				return fmt.Errorf("Error deleting file from registry: %v", err)
			}
			r.log.Debugln("File deleted from the registry")
		}
	}

	r.log.Debugln("Transactions of block updated to the registry")
	return nil
}
