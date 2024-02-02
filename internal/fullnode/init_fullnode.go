package fullnode

import (
	"context"
	"fmt"
	"os"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
)

// Verify if the blockchain needs be fetched from the network
func NeedToFetchBlockchain(ctx context.Context) bool {
	// Check if the blockchain exists
	blockChainInfo, err := os.Stat("./blockchain")

	if os.IsNotExist(err) {
		//fmt.Println("Blockchain doesn't exist")
		return true

	} else {

		fmt.Println("Blockchain exists")
		lastTimeModified := blockChainInfo.ModTime()
		currentTime := time.Now()

		if currentTime.Sub(lastTimeModified) > 1*time.Hour {
			// If the blockchain has not been updated for more than an hour, it means that the node is not up to date
			fmt.Println("Blockchain is not up to date")
			return true

		} else {
			fmt.Println("Blockchain is up to date")
			return false

		}
	}

}

// Ask the network for the blockchain
func FetchBlockchain(ctx context.Context, timeout time.Duration, ps *pubsub.PubSub) *pebble.PebbleTransactionDB {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Join the topic FullNodeAskingForBlockchain
	fullNodeAskingForBlockchainTopic, err := ps.Join("FullNodeAskingForBlockchain")
	if err != nil {
		panic(err)
	}

	// Subscribe to the topic FullNodeGivingBlockchain
	fullNodeGivingBlockchainTopic, err := ps.Join("FullNodeGivingBlockchain")
	if err != nil {
		panic(err)
	}
	subFullNodeGivingBlockchain, err := fullNodeGivingBlockchainTopic.Subscribe()
	if err != nil {
		panic(err)
	}

	blockchainReceive := make(chan bool)
	go func() {
		for {
			msg, _ := subFullNodeGivingBlockchain.Next(ctx)
			// if err != nil {
			// 	fmt.Printf("Error getting blockchain announcement message : %s", err)
			// 	continue
			// }

			if msg != nil {
				fmt.Println("Blockchain: ", string(msg.Data))
				blockchainReceive <- true
				break
			}

		}

	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Timeout reached, creating a new blockchain database with Pebble")
			database := createDatabase()
			return database

		case <-blockchainReceive:
			fmt.Println("Blockchain successfully received. Exiting request loop.")
			//process to pull the blockchain with ipfs ...
			return nil

		case <-ticker.C:
			fmt.Println("Requesting blockchain from the network")

			err = fullNodeAskingForBlockchainTopic.Publish(ctx, []byte("I need the blockchain"))
			if err != nil {
				fmt.Printf("Error publishing blockchain announcement to the network : %s", err)
			}

		}
	}
}

// Create a database with Pebble db
func createDatabase() *pebble.PebbleTransactionDB {
	pebbleDB, err := pebble.NewPebbleTransactionDB("blockchain")
	if err != nil {
		fmt.Printf("Error creating database: %s", err)
		return nil
	}

	return pebbleDB
}
