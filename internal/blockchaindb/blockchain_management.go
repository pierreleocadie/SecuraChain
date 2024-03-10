package blockchaindb

import (
	"context"
	"fmt"

	icore "github.com/ipfs/kubo/core/coreiface"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

func AskTheBlockchainRegistry(ctx context.Context, ps *pubsub.PubSub) ([]byte, string, error) {
	// Join the topic to ask for the json file of the blockchain
	fullNodeAskingForBlockchainTopic, err := ps.Join("FullNodeAskingForBlockchain")
	if err != nil {
		return nil, "", fmt.Errorf("error joining FullNodeAskingForBlockchain topic %s", err)
	}

	// Publish a message to ask for the blockchain
	fmt.Println("Requesting blockchain from the network")
	if err := fullNodeAskingForBlockchainTopic.Publish(ctx, []byte("I need the json file of your blockchain")); err != nil {
		return nil, "", fmt.Errorf("error publishing blockchain request %s", err)
	}

	// Join the topic to receive the json file of the blockchain
	fullNodeGivingBlockchainTopic, err := ps.Join("FullNodeGivingBlockchain")
	if err != nil {
		return nil, "", fmt.Errorf("error joining FullNodeGivingBlockchain topic %s", err)
	}
	// Subscribe to the topic to receive the blockchain
	subFullNodeGivingBlockchain, err := fullNodeGivingBlockchainTopic.Subscribe()
	if err != nil {
		return nil, "", fmt.Errorf("error subscribing to FullNodeGivingBlockchain topic %s", err)
	}

	jsonfile := make(chan []byte)
	sender := make(chan string)
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
				sender <- msg.GetFrom().String()
				break
			}
		}
	}()

	return <-jsonfile, <-sender, nil

}

func IntegrityAndUpdate(ctx context.Context, ipfsAPI icore.CoreAPI, ps *pubsub.PubSub, database *BlockchainDB) bool {
	// create a list of blacklisted nodes
	blackListNode := []string{}

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
			updated, sender, err := DownloadMissingBlocks(ctx, ipfsAPI, blackListNode, ps, database)
			if err != nil {
				fmt.Println("Error downloading missing blocks : ", err)
				return false
			}
			if !updated {
				fmt.Println("Error downloading missing blocks and verifying the blockchain")
				blackListNode = append(blackListNode, sender)
				continue
			}
		} else {
			break // the blockchain is verified
		}
	}
	return true
}

func DownloadMissingBlocks(ctx context.Context, ipfsAPI icore.CoreAPI, blackListNode []string, ps *pubsub.PubSub, database *BlockchainDB) (bool, string, error) {

	registryBytes, sender, err := AskTheBlockchainRegistry(ctx, ps)
	if err != nil {
		return false, "", fmt.Errorf("error asking the blockchain registry : %s", err)
	}

	// Check if the sender is blacklisted
	if nodeBlackListed(blackListNode, sender) {
		return false, "", fmt.Errorf("node blacklisted")
	}

	// Convert the regustryBytes to blockRegistry
	registry, err := ConvertByteToBlockRegistry(registryBytes)
	if err != nil {
		return false, "", fmt.Errorf("error converting bytes to block registry : %s", err)
	}

	for _, block := range registry.Blocks {
		// Get the block from IPFS
		blockIPFS, err := GetBlockFromIPFS(ctx, ipfsAPI, block.Cid)
		if err != nil {
			return false, "", fmt.Errorf("error getting block from IPFS : %s", err)
		}

		// Add the block to the blockchain
		added, message := AddBlockToBlockchain(blockIPFS, database)
		fmt.Println(message)
		if !added {
			if message == "Block already in the blockchain" {
				continue // on passe au block suivant
			} else {
				return false, "", fmt.Errorf("error adding block to blockchain")
			}
		}

	}

	integrity, err := database.VerifyBlockchainIntegrity(database.GetLastBlock())
	if err != nil {
		return false, "", fmt.Errorf("error verifying blockchain integrity : %s", err)
	}

	if !integrity {
		return false, "", fmt.Errorf("error verifying blockchain integrity")
	}

	return true, sender, nil

}

func nodeBlackListed(blackListNode []string, sender string) bool {
	blacklisted := false
	for _, node := range blackListNode {
		if node == sender {
			blacklisted = true
			break
		}
	}
	return blacklisted
}
