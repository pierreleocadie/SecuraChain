package storagenode

import (
	"os"
	"path/filepath"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	fileregistry "github.com/pierreleocadie/SecuraChain/internal/registry/file_registry"
)

type StorageGarbageCollector struct {
	fileRegistry fileregistry.FileRegistry
	log          *ipfsLog.ZapEventLogger
	config       *config.Config
}

func NewStorageGarbageCollector(log *ipfsLog.ZapEventLogger, config *config.Config, fileRegistry fileregistry.FileRegistry) *StorageGarbageCollector {
	return &StorageGarbageCollector{
		fileRegistry: fileRegistry,
		log:          log,
		config:       config,
	}
}

func (s *StorageGarbageCollector) Update(state string) {
	s.log.Debugln("Garbage collector state updated: ", state)
	go s.ProcessGarbageCollection()
}

func (s *StorageGarbageCollector) ProcessGarbageCollection() {
	s.log.Debugln("Processing garbage collection")

	cidsString := []string{}

	// 1. Get all CID from the file registry
	for _, files := range s.fileRegistry.GetRegistry() {
		for _, file := range files {
			cidsString = append(cidsString, file.FileCid.String())
		}
	}

	// 2. Remove duplicates from the list of CIDs
	uniqueCids := make(map[string]struct{})
	for _, cid := range cidsString {
		uniqueCids[cid] = struct{}{}
	}

	cidsString = []string{} // Reset the slice
	for cid := range uniqueCids {
		cidsString = append(cidsString, cid) // Rebuild the slice without duplicates
	}

	// 3. Get all file names from the IPFS local storage directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		s.log.Errorln("Error getting the user home directory : ", err)
		return
	}

	ipfsStoragePath := filepath.Join(homeDir, ".IPFS_Local_Storage")

	files, err := os.ReadDir(ipfsStoragePath)
	if err != nil {
		s.log.Errorln("Error reading the IPFS local storage directory : ", err)
		return
	}

	// 4. Compare the list of CIDs with the list of files in the IPFS local storage directory
	filesExist := make(map[string]bool)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Check if the file is a CID
		for _, cid := range cidsString {
			if file.Name() == cid {
				s.log.Debugln("File found in the IPFS local storage directory: ", file.Name())
				filesExist[file.Name()] = true
				break
			}
		}
	}

	// 5. Remove the files that are not in the list of CIDs
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if _, ok := filesExist[file.Name()]; !ok {
			s.log.Debugln("File not found in the file registry: ", file.Name())
			filePath := filepath.Join(ipfsStoragePath, file.Name())
			if err := os.Remove(filePath); err != nil {
				s.log.Errorln("Error deleting file: ", err)
			}
		}
	}

}
