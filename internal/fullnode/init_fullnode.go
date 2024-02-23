package fullnode

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
)

// NeedToFet@chBlockchain checks if the blockchain exists and if it is up to date
func HasABlockchain() bool {
	blockChainInfo, err := os.Stat("./blockchain")

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

// FetchBlockchain requests the blockchain from the network or creates a new one if not received.
func FetchBlockchain(ctx context.Context, ipfsAPI icore.CoreAPI, timeout time.Duration, ps *pubsub.PubSub) *pebble.PebbleTransactionDB {
	var interval = 30 * time.Second

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Join the topic to ask for the blockchain
	fullNodeAskingForBlockchainTopic, err := ps.Join("FullNodeAskingForBlockchain")
	if err != nil {
		fmt.Println("Error joining FullNodeAskingForBlockchain topic : ", err)
		return nil
	}

	// Join the topic to receive the blockchain
	fullNodeGivingBlockchainTopic, err := ps.Join("FullNodeGivingBlockchain")
	if err != nil {
		fmt.Println("Error joining to FullNodeGivingBlockchain topic : ", err)
		return nil
	}
	// Subscribe to the topic to receive the blockchain
	subFullNodeGivingBlockchain, err := fullNodeGivingBlockchainTopic.Subscribe()
	if err != nil {
		fmt.Println("Error subscribing to FullNodeGivingBlockchain topic : ", err)
		return nil
	}

	blockchainReceive := make(chan bool)
	go func() {
		for {
			msg, _ := subFullNodeGivingBlockchain.Next(ctx)
			if msg != nil {
				fmt.Println("Blockchain received from the network")
				blockchainReceive <- true

				cidStr := string(msg.Data)
				newPath, err := path.NewPath(cidStr)
				if err != nil {
					fmt.Printf("Error parsing CID to path : %s\n", err)
				}

				cidDirectory, err := path.NewImmutablePath(newPath)
				if err != nil {
					fmt.Printf("Error parsing CID to path : %s\n", err)
				}

				// Get the blockchain from IPFS
				if err := ipfs.GetDirectoryWithPath(ctx, ipfsAPI, cidDirectory, "./blockchain"); err != nil {
					fmt.Printf("Error getting blockchain from IPFS : %s\n", err)
				}
				break
			}
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Timeout reached, creating a new blockchain database with Pebble")
		return createDatabase()
	case <-blockchainReceive:
		fmt.Println("Blockchain successfully received. Exiting request loop.")
		//process to pull the blockchain with ipfs ...
		return nil
	case <-ticker.C:
		fmt.Println("Requesting blockchain from the network")
		if err := fullNodeAskingForBlockchainTopic.Publish(ctx, []byte("I need the blockchain")); err != nil {
			fmt.Printf("Error publishing blockchain request : %s\n", err)
		}
	}
	return nil
}

// createDatabase initializes a new Pebble database for the blockchain.
func createDatabase() *pebble.PebbleTransactionDB {
	pebbleDB, err := pebble.NewPebbleTransactionDB("blockchain")
	if err != nil {
		fmt.Printf("Error creating database: %s", err)
		return nil
	}

	return pebbleDB
}
