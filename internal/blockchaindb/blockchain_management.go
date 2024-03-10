package blockchaindb

import (
	"context"
	"fmt"

	icore "github.com/ipfs/kubo/core/coreiface"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// // HasABlockchain checks if the blockchain exists and if it is up to date
// func HasABlockchain() bool {
// 	blockChainInfo, err := os.Stat("blockchain")

// 	if os.IsNotExist(err) {
// 		fmt.Println("Blockchain doesn't exist")
// 		return false
// 	}

// 	fmt.Println("Blockchain exists")
// 	lastTimeModified := blockChainInfo.ModTime()
// 	currentTime := time.Now()

// 	// if the blockchain has not been modified for more than 1 hour, we need to fetch the blockchain.
// 	if currentTime.Sub(lastTimeModified) > 1*time.Hour {
// 		fmt.Println("Blockchain is not up to date")
// 		return false
// 	}

// 	fmt.Println("Blockchain is up to date")
// 	return true
// }

func AskTheBlockchainRegistry(ctx context.Context, ps *pubsub.PubSub) ([]byte, error) {
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

// func DownloadBlockchain(ctx context.Context, ipfsAPI icore.CoreAPI, cidBlock string, jsonfile string) (*PebbleDB, error) {
// 	if !HasABlockchain() {
// 		blockchain, err := NewBlockchainDB("blockchain")
// 		if err != nil {
// 			fmt.Println("Error creating blockchain database : ", err)
// 		}

// 		blockRegister, err := ReadBlockDataFromFile(jsonfile)
// 		if err != nil {
// 			fmt.Println("Error reading block data from file : ", err)
// 		}

// 		for {
// 			for _, block := range blockRegister.Blocks {
// 				// Get the block from IPFS
// 				blockIPFS, err := ipfs.GetBlock(ctx, ipfsAPI, block.Cid)
// 				if err != nil {
// 					fmt.Println("Error getting block from IPFS : ", err)
// 				}

// 				// Add the block to the blockchain
// 				added, message := AddBlockToBlockchain(blockIPFS, blockchain)
// 				fmt.Println(message)
// 				if !added {
// 					fmt.Println("Error adding block to blockchain")
// 				}

// 			}

// 			lastBlock := blockchain.GetLastBlock()
// 			integrity, err := blockchain.VerifyBlockchainIntegrity(lastBlock)
// 			if err != nil {
// 				fmt.Println("Error verifying blockchain integrity : ", err)
// 			}
// 			if !integrity {
// 				fmt.Println("Blockchain integrity compromised")
// 			}
// 			break
// 		}
// 		return blockchain, nil
// 	} else {
// 		blockchain, err := NewBlockchainDB("blockchain")
// 		if err != nil {
// 			fmt.Println("Error creating blockchain database : ", err)
// 		}
// 		// S'il a une blockchain,
// 		//on compare le fichier de la nouvelle blockchaain avec le notre et
// 		//on met à jour le notre blockchain en vérifiant qu'elle soit toujorus intègre
// 		return blockchain, nil
// 	}
// }

func IntegrityAndUpdate(ctx context.Context, ipfsAPI icore.CoreAPI, ps *pubsub.PubSub, database *BlockchainDB) bool {
	for {
		lastBlock := database.GetLastBlock()
		if lastBlock == nil {
			fmt.Println("No block in the blockchain")
			return true
		}

		integrity, err := database.VerifyBlockchainIntegrity(lastBlock)
		if err != nil {
			return false
		}

		if !integrity {
			fmt.Println("Blockchain not verified")
			// Download the missing blocks and verify the blockchain
			updated, err := DownloadMissingBlocks(ctx, ipfsAPI, ps, database)
			if err != nil {
				fmt.Println("Error downloading missing blocks : ", err)
				return false
			}
			if !updated {
				fmt.Println("Error downloading missing blocks and verifying the blockchain")
				continue
			}
		} else {
			break // the blockchain is verified
		}
	}
	return true
}

func DownloadMissingBlocks(ctx context.Context, ipfsAPI icore.CoreAPI, ps *pubsub.PubSub, database *BlockchainDB) (bool, error) {

	registryBytes, err := AskTheBlockchainRegistry(ctx, ps)
	if err != nil {
		fmt.Println("Error asking the blockchain registry : ", err)
		return false, err
	}

	// Convert the regustryBytes to blockRegistry
	registry, err := ConvertByteToBlockRegistry(registryBytes)
	if err != nil {
		fmt.Println("Error converting bytes to block registry : ", err)
		return false, err
	}

	for _, block := range registry.Blocks {
		// Get the block from IPFS
		blockIPFS, err := GetBlockFromIPFS(ctx, ipfsAPI, block.Cid)
		if err != nil {
			fmt.Println("Error getting block from IPFS : ", err)
		}

		// Add the block to the blockchain
		added, message := AddBlockToBlockchain(blockIPFS, database)
		fmt.Println(message)
		if !added {
			if message == "Block already in the blockchain" {
				continue // on passe au block suivant
			} else {
				return false, fmt.Errorf("error adding block to blockchain")
			}
		}

	}

	integrity, err := database.VerifyBlockchainIntegrity(database.GetLastBlock())
	if err != nil {
		return false, err
	}

	if !integrity {
		return false, fmt.Errorf("blockchain integrity compromised")
	}

	return true, nil

}
