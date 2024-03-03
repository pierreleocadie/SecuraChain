package fullnode_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func TestPublishBlockToIPFS(t *testing.T) {
	t.Parallel()

	// Create a new PebbleTransactionDB instance
	database, err := pebble.NewBlockchainDB("blockchain")
	if err != nil {
		t.Fatalf("Error creating PebbleTransactionDB: %v", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	/*
	* FIRST BLOCK
	 */

	// Create a genesis block
	genesisBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)
	genesisBlockBytes, err := genesisBlock.Serialize()
	if err != nil {
		t.Errorf("Error serializing block: %v", err)
	}

	// Check if the returned block is not nil
	if genesisBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Call the AddBlockToBlockchain function
	added, message := pebble.AddBlockToBlockchain(genesisBlock, database)

	// Verify the result
	if !added {
		t.Errorf("Expected block to be added, but it was not added")
	}
	expectedMessage := "Block addded succesfully to the blockchain"
	if message != expectedMessage {
		t.Errorf("Expected message: %s, but got: %s", expectedMessage, message)
	}

	ctx := context.Background()

	cfg, err := config.LoadConfig("./config-test.yml")
	if err != nil {
		t.Errorf("Error loading config file : %v", err)
	}

	ipfsAPI, nodeIpfs, err := ipfs.SpawnNode(ctx, cfg)
	if err != nil {
		t.Errorf("Error spawning IPFS node : %v", err)
	}

	if err := fullnode.PublishBlockToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, genesisBlock); err != nil {
		t.Errorf("Error publishing block to IPFS : %v", err)
	}

	blockRegister, err := pebble.ReadBlockDataFromFile("blocksRegistry.json")
	if err != nil {
		t.Errorf("Error reading block data from file : %v", err)
	}

	newCidBlock := blockRegister.Blocks[0].Cid

	// Get the block from IPFS
	blockIPFS, err := ipfs.GetBlock(ctx, cfg, ipfsAPI, newCidBlock, "block1")
	if err != nil {
		t.Errorf("Error getting block from IPFS : %v", err)
	}

	blockDownloaded, err := blockIPFS.Serialize()
	if err != nil {
		t.Errorf("Error serializing block : %v", err)
	}

	if !bytes.Equal(genesisBlockBytes, blockDownloaded) {
		t.Errorf("Expected block to be equal, but it was not equal")
	}

}
