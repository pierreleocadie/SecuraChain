package pebble

import (
	"fmt"
	"os"
	"time"
)

// HasABlockchain checks if the blockchain exists and if it is up to date
func HasABlockchain() bool {
	blockChainInfo, err := os.Stat("blockchain")

	if os.IsNotExist(err) {
		fmt.Println("Blockchain doesn't exist")
		return false
	}

	fmt.Println("Blockchain exists")
	lastTimeModified := blockChainInfo.ModTime()
	currentTime := time.Now()

	// if the blockchain has not been modified for more than 1 hour, we need to fetch the blockchain.
	if currentTime.Sub(lastTimeModified) > 1*time.Hour {
		fmt.Println("Blockchain is not up to date")
		return false
	}

	fmt.Println("Blockchain is up to date")
	return true
}

func DownloadBlockchain() {
	if !HasABlockchain() {
		for {

		}
	}
}
