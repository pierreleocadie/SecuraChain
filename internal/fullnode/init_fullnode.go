package fullnode

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/boxo/path"
	icore "github.com/ipfs/kubo/core/coreiface"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
)

// DownloadBlockchain requests the blockchain from the network.
func DownloadBlockchain(ctx context.Context, ipfsAPI icore.CoreAPI, ps *pubsub.PubSub) (bool, error, string) {
	var interval = 30 * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Join the topic to ask for the blockchain
	fullNodeAskingForBlockchainTopic, err := ps.Join("FullNodeAskingForBlockchain")
	if err != nil {
		fmt.Println("Error joining FullNodeAskingForBlockchain topic : ", err)
		return false, err, ""
	}

	// Join the topic to receive the blockchain
	fullNodeGivingBlockchainTopic, err := ps.Join("FullNodeGivingBlockchain")
	if err != nil {
		fmt.Println("Error joining to FullNodeGivingBlockchain topic : ", err)
		return false, err, ""
	}
	// Subscribe to the topic to receive the blockchain
	subFullNodeGivingBlockchain, err := fullNodeGivingBlockchainTopic.Subscribe()
	if err != nil {
		fmt.Println("Error subscribing to FullNodeGivingBlockchain topic : ", err)
		return false, err, ""
	}

	blockchainReceive := make(chan bool)
	sender := make(chan string)
	go func() {
		for {
			msg, _ := subFullNodeGivingBlockchain.Next(ctx)
			if msg != nil {
				fmt.Println("Blockchain received from the network")
				blockchainReceive <- true

				// Get the sender of the message
				sender <- msg.GetFrom().String()

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
	senderBlockchain := <-sender

	select {
	case <-blockchainReceive:
		fmt.Println("Blockchain successfully received. Exiting request loop.")
		return true, nil, senderBlockchain
	case <-ticker.C:
		fmt.Println("Requesting blockchain from the network")
		if err := fullNodeAskingForBlockchainTopic.Publish(ctx, []byte("I need the blockchain")); err != nil {
			fmt.Printf("Error publishing blockchain request : %s\n", err)
		}
	}
	return false, nil, ""
}

// IsGenesisBlock checks if the block is the genesis block
func IsGenesisBlock(b *block.Block) bool {
	if b.PrevBlock == nil && b.Header.Height == 1 {
		return true
	}
	return false
}
