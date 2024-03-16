package blockchaindb

import (
	"context"
	"fmt"

	icore "github.com/ipfs/kubo/core/coreiface"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

func AskTheBlockchainRegistry(ctx context.Context, askingBlockchain *pubsub.Topic, receiveBlockchain *pubsub.Subscription) ([]byte, string, error) {
	// Publish a message to ask for the blockchain
	fmt.Println("Requesting blockchain from the network")
	if err := askingBlockchain.Publish(ctx, []byte("I need the json file of your blockchain")); err != nil {
		return nil, "", fmt.Errorf("error publishing blockchain request %s", err)
	}

	jsonfile := make(chan []byte)
	sender := make(chan string)
	go func() {
		for {
			msg, err := receiveBlockchain.Next(ctx)
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

func DownloadMissingBlocks(ctx context.Context, ipfsAPI icore.CoreAPI, registry []byte, database *BlockchainDB) (bool, []*block.Block, error) {
	var listOfMissingBlocks []*block.Block
	dhtAPI := ipfsAPI.Dht()
	// Convert the regustryBytes to blockRegistry
	registryBlockchain, err := ConvertByteToBlockRegistry(registry)
	if err != nil {
		return false, nil, fmt.Errorf("error converting bytes to block registry : %s", err)
	}

	for _, block := range registryBlockchain.Blocks {
		// Check if the block is already in the blockchain
		b, err := database.GetBlock(block.Key)
		if err == nil && b != nil {
			continue // we go to the next block, because the block is already in the blockchain
		}

		// -------
		providers, err := dhtAPI.FindProviders(ctx, block.Cid)
		if err != nil {
			fmt.Println("error finding providers : ", err)
			continue
		}
	outer:
		for {
			providers, err := dhtAPI.FindProviders(ctx, block.Cid)
			if err != nil {
				fmt.Println("error finding providers : ", err)
				continue
			}
			for provider := range providers {
				fmt.Println("found provider : ", provider.ID.String())
				break outer
			}
			fmt.Printf("Channel contains %d providers", len(providers))
			if len(providers) >= 1 {
				break
			}
		}
		for provider := range providers {
			err := ipfsAPI.Swarm().Connect(ctx, provider)
			if err != nil {
				fmt.Printf("failed to connect to provider: %s", err)
				continue
			}
		}
		fmt.Printf("Downloading file %s", block.Cid.String())
		// 	//--------
		// Get the block from IPFS
		blockIPFS, err := GetBlockFromIPFS(ctx, ipfsAPI, block.Cid)
		if err != nil {
			return false, nil, fmt.Errorf("error getting block from IPFS : %s", err)
		}

		listOfMissingBlocks = append(listOfMissingBlocks, blockIPFS)
	}
	return true, listOfMissingBlocks, nil
}

// func IntegrityAndUpdate(ctx context.Context, ipfsAPI icore.CoreAPI, ps *pubsub.PubSub, database *BlockchainDB) bool {
// 	// create a list of blacklisted nodes
// 	blackListNode := []string{}

// 	for {
// 		lastBlock := database.GetLastBlock()
// 		if lastBlock == nil {
// 			fmt.Println("No block in the blockchain")
// 			return true
// 		}

// 		integrity, err := database.VerifyBlockchainIntegrity(lastBlock)
// 		if err != nil {
// 			return false
// 		}

// 		if !integrity {
// 			fmt.Println("Blockchain not verified")
// 			// Download the missing blocks and verify the blockchain
// 			updated, sender, err := DownloadMissingBlocks(ctx, ipfsAPI, blackListNode, ps, database)
// 			if err != nil {
// 				fmt.Println("Error downloading missing blocks : ", err)
// 				return false
// 			}
// 			if !updated {
// 				fmt.Println("Error downloading missing blocks and verifying the blockchain")
// 				blackListNode = append(blackListNode, sender)
// 				continue
// 			}
// 		} else {
// 			break // the blockchain is verified
// 		}
// 	}
// 	return true
// }

// func convertStringToPathImmutable(stringCid string) (path.ImmutablePath, error) {
// 	c, err := cid.Decode(stringCid)
// 	if err != nil {
// 		return path.ImmutablePath{}, fmt.Errorf("error decoding the path : %s", err)
// 	}

// 	pathImmutable := path.FromCid(c)

// 	return pathImmutable, nil
// }
