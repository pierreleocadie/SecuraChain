package pebble

import (
	"context"
	"fmt"
	"os"
	"time"

	icore "github.com/ipfs/kubo/core/coreiface"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
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

func AskForABlockchain(ctx context.Context, ps *pubsub.PubSub) ([]byte, error) {
	// Join the topic to ask for the json file of the blockchain
	fullNodeAskingForBlockchainTopic, err := ps.Join("FullNodeAskingForBlockchain")
	if err != nil {
		fmt.Println("Error joining FullNodeAskingForBlockchain topic : ", err)
		return nil, err
	}

	// Publish a message to ask for the blockchain
	fmt.Println("Requesting blockchain from the network")
	if err := fullNodeAskingForBlockchainTopic.Publish(ctx, []byte("I need the json file of your blockchain")); err != nil {
		fmt.Printf("Error publishing blockchain request : %s\n", err)
	}

	// Join the topic to receive the json file of the blockchain
	fullNodeGivingBlockchainTopic, err := ps.Join("FullNodeGivingBlockchain")
	if err != nil {
		fmt.Println("Error joining to FullNodeGivingBlockchain topic : ", err)
		return nil, err
	}
	// Subscribe to the topic to receive the blockchain
	subFullNodeGivingBlockchain, err := fullNodeGivingBlockchainTopic.Subscribe()
	if err != nil {
		fmt.Println("Error subscribing to FullNodeGivingBlockchain topic : ", err)
		return nil, err
	}

	jsonfile := make(chan []byte)
	go func() {
		for {
			msg, err := subFullNodeGivingBlockchain.Next(ctx)
			if err != nil {
				fmt.Println("Error getting message from the network : ", err)
				break
			}
			if msg != nil {
				fmt.Println("Blockchain received from the network")
				jsonfile <- msg.Data
				break
			}
		}
	}()

	return <-jsonfile, nil

}

func DownloadBlockchain(ctx context.Context, ipfsAPI icore.CoreAPI, cidBlock string, jsonfile string) (*PebbleDB, error) {
	if !HasABlockchain() {
		blockchain, err := NewBlockchainDB("blockchain")
		if err != nil {
			fmt.Println("Error creating blockchain database : ", err)
		}

		blockRegister, err := ReadBlockDataFromFile(jsonfile)
		if err != nil {
			fmt.Println("Error reading block data from file : ", err)
		}

		for {
			for _, block := range blockRegister.Blocks {
				// Get the block from IPFS
				blockIPFS, err := ipfs.GetBlock(ctx, ipfsAPI, block.Cid)
				if err != nil {
					fmt.Println("Error getting block from IPFS : ", err)
				}

				// Add the block to the blockchain
				added, message := AddBlockToBlockchain(blockIPFS, blockchain)
				fmt.Println(message)
				if !added {
					fmt.Println("Error adding block to blockchain")
				}

			}

			lastBlock := blockchain.GetLastBlock()
			integrity, err := blockchain.VerifyBlockchainIntegrity(lastBlock)
			if err != nil {
				fmt.Println("Error verifying blockchain integrity : ", err)
			}
			if !integrity {
				fmt.Println("Blockchain integrity compromised")
			}
			break
		}
		return blockchain, nil
	} else {
		blockchain, err := NewBlockchainDB("blockchain")
		if err != nil {
			fmt.Println("Error creating blockchain database : ", err)
		}
		// S'il a une blockchain,
		//on compare le fichier de la nouvelle blockchaain avec le notre et
		//on met à jour le notre blockchain en vérifiant qu'elle soit toujorus intègre
		return blockchain, nil
	}
}

func IntegrityAndUpdate(database *PebbleDB) bool {
	for {
		lastBlock := database.GetLastBlock()

		integrity, err := database.VerifyBlockchainIntegrity(lastBlock)
		if err != nil {
			return false
		}

		if !integrity {
			fmt.Prinln("Blockchain not verified")
			continue
			// on demande la blockchain pour se mettre à jour
			// et on re test l'intégrité
		}
		break // the blockchain is verified
	}
	return true
}
