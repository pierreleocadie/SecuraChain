package ipfs

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ipfs/kubo/core"
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
func ChangeStorageMax(nodeIpfs *core.IpfsNode, memorySpace uint) (bool, error) {
	// Get the config file of the IPFS node
	configFileIPFS, err := nodeIpfs.Repo.Config()
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
	if err := nodeIpfs.Repo.SetConfig(configFileIPFS); err != nil {
		return false, fmt.Errorf("failed to set IPFS config: %s", err)
	}

	// Get the config file of the IPFS node to verify new StorageMax
	configFileIPFS2, err := nodeIpfs.Repo.Config()
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

func FreeMemoryAvailable(ctx context.Context, nodeIpfs *core.IpfsNode) (uint64, error) {
	spaceUsed, err := nodeIpfs.Repo.GetStorageUsage(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get the number of bytes stored: %s", err)
	}

	configFileIPFS, err := nodeIpfs.Repo.Config()
	if err != nil {
		return 0, fmt.Errorf("failed to get IPFS config: %s", err)
	}

	storageMax, err := strconv.ParseUint(configFileIPFS.Datastore.StorageMax[:len(configFileIPFS.Datastore.StorageMax)-2], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to convert storageMax string to int: %s", err)
	}

	storageMax = storageMax * GB

	freeMemoryAvailable := storageMax - spaceUsed

	return freeMemoryAvailable, nil
}

func MemoryUsed(ctx context.Context, nodeIpfs *core.IpfsNode) (uint64, error) {
	spaceUsed, err := nodeIpfs.Repo.GetStorageUsage(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get the number of bytes stored: %s", err)
	}

	return spaceUsed, nil
}
