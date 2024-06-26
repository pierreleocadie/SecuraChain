package ipfs

import (
	"fmt"
	"strconv"
)

const (
	B  = 1
	KB = 1000 * B
	MB = 1000 * KB
	GB = 1000 * MB
)

/*
* MEMORY SHARE TO THE BLOCKCHAIN BY THE NODE
 */
func (ipfs *IPFSNode) ChangeStorageMax(memorySpace uint) (bool, error) {
	// Get the config file of the IPFS node
	configFileIPFS, err := ipfs.Node.Repo.Config()
	if err != nil {
		return false, fmt.Errorf("failed to get IPFS config: %s", err)
	}

	// Before update the storageMax let's check if the new value is different from the current one
	storageMax, err := strconv.ParseUint(configFileIPFS.Datastore.StorageMax[:len(configFileIPFS.Datastore.StorageMax)-2], 10, 64)
	if err != nil {
		return false, fmt.Errorf("failed to convert storageMax string to int: %s", err)
	}

	if storageMax == uint64(memorySpace) {
		return false, nil
	}

	// Update the StorageMax
	configFileIPFS.Datastore.StorageMax = fmt.Sprintf("%dGB", memorySpace)
	if err := ipfs.Node.Repo.SetConfig(configFileIPFS); err != nil {
		return false, fmt.Errorf("failed to set IPFS config: %s", err)
	}

	// Get the config file of the IPFS node to verify new StorageMax
	configFileIPFS2, err := ipfs.Node.Repo.Config()
	if err != nil {
		return false, fmt.Errorf("failed to get IPFS config: %s", err)
	}

	newStorageMax, err := strconv.ParseUint(configFileIPFS2.Datastore.StorageMax[:len(configFileIPFS2.Datastore.StorageMax)-2], 10, 64)
	if err != nil {
		return false, fmt.Errorf("failed to convert string to int: %s", err)
	}

	if newStorageMax != uint64(memorySpace) {
		return false, fmt.Errorf("failed to set new StorageMax. Expected %dGB, got %dGB", memorySpace, newStorageMax)
	}

	return true, nil
}

func (ipfs *IPFSNode) FreeMemoryAvailable() (uint64, error) {
	spaceUsed, err := ipfs.Node.Repo.GetStorageUsage(ipfs.Ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get the number of bytes stored: %s", err)
	}

	configFileIPFS, err := ipfs.Node.Repo.Config()
	if err != nil {
		return 0, fmt.Errorf("failed to get IPFS config: %s", err)
	}

	storageMax, err := strconv.ParseUint(configFileIPFS.Datastore.StorageMax[:len(configFileIPFS.Datastore.StorageMax)-2], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to convert storageMax string to int: %s", err)
	}

	storageMax *= GB

	freeMemoryAvailable := storageMax - spaceUsed

	return freeMemoryAvailable, nil
}

func (ipfs *IPFSNode) MemoryUsed() (uint64, error) {
	spaceUsed, err := ipfs.Node.Repo.GetStorageUsage(ipfs.Ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get the number of bytes stored: %s", err)
	}

	return spaceUsed, nil
}
