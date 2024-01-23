package ipfs

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ipfs/kubo/core"
)

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

/*
* MEMORY SHARE TO THE BLOCKCHAIN BY THE NODE
 */

func ChangeStorageMax(nodeIpfs *core.IpfsNode) (int, error) {
	// Change the storage max
	var memory int
	fmt.Println("\nCombien d'espaces mémoire voulez-vous allouer ? (en Go)")

	_, err := fmt.Scan(&memory)
	if err != nil {
		return 0, fmt.Errorf("failed to scan memory: %s", err)
	}

	for err != nil || memory < 0 {
		fmt.Println("\nEntrez un nombre entier positif ? (en Go)")
		fmt.Scan(&memory)
	}

	fmt.Println("Vous avez alloué ", memory, "Go d'espace mémoire")
	// fmt.Println("Storage max set to ", memory, "Go")

	// Get the config file of the IPFS node
	configFileIPFS, err := nodeIpfs.Repo.Config()
	if err != nil {
		return 0, fmt.Errorf("failed to get IPFS config: %s", err)
	}

	// fmt.Println("StorageMax is : ", configFileIPFS.Datastore.StorageMax)

	configFileIPFS.Datastore.StorageMax = fmt.Sprintf("%dGB", memory)
	if err := nodeIpfs.Repo.SetConfig(configFileIPFS); err != nil {
		return 0, fmt.Errorf("failed to set IPFS config: %s", err)
	}

	// Get the config file of the IPFS node to verify new StorageMax
	configFileIPFS2, err := nodeIpfs.Repo.Config()
	if err != nil {
		return 0, fmt.Errorf("failed to get IPFS config: %s", err)
	}

	configFileIPFS2.Datastore.StorageMax = configFileIPFS2.Datastore.StorageMax[:len(configFileIPFS2.Datastore.StorageMax)-2]
	// fmt.Println("New StorageMax is : ", configFileIPFS2.Datastore.StorageMax)
	newStorageMax, err := strconv.Atoi(configFileIPFS2.Datastore.StorageMax)
	if err != nil {
		return 0, fmt.Errorf("failed to convert storage max to int: %s", err)
	}

	return newStorageMax, nil
}

func FreeMemoryAvailable(ctx context.Context, nodeIpfs *core.IpfsNode, storageMax int) (float64, error) {
	// returns the number of bytes stores
	sizeTaken, err := nodeIpfs.Repo.GetStorageUsage(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get the number of bytes stored: %s", err)
	}

	// fmt.Println("Nombre de bytes stockés : ", float64(sizeTaken))
	memoryGB := float64(sizeTaken) / float64(GB) // Nombre de bytes stockeés en Gigabytes
	// memoryMB := float64(sizeTaken) / float64(MB) // Nombre de bytes stockeés en Megabytes

	fmt.Println("Memory available in Gigabytes : ", float64(storageMax)-memoryGB)
	// fmt.Println("Memory available in MegaBytes : ", float64(storageMax)-memoryMB)

	freeSpaceAvailableGB := float64(storageMax) - memoryGB
	// freeSpaceAvailableMB := float64(storageMax) - memoryMB

	return freeSpaceAvailableGB, nil
}

// returns the number of bytes stored in GigaBytes and MegaBytes
func MemoryUsed(ctx context.Context, nodeIpfs *core.IpfsNode) (float64, error) {
	sizeTaken, err := nodeIpfs.Repo.GetStorageUsage(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get the number of bytes stored: %s", err)
	}

	// fmt.Println("Nombre de bytes stockés : ", float64(sizeTaken))
	memoryGB := float64(sizeTaken) / float64(GB) // Nombre de bytes stockeés en Gigabytes
	// memoryMB := float64(sizeTaken) / float64(MB) // Nombre de bytes stockeés en Megabytes

	// fmt.Println("Memory used in GigaBytes : ", memoryGB)
	// fmt.Println("Memory used in MegaBytes : ", memoryMB)

	return memoryGB, nil
}
