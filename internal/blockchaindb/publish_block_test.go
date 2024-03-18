// package blockchaindb_test

// import (
// 	"bytes"
// 	"context"
// 	"os"
// 	"testing"

// 	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
// 	"github.com/pierreleocadie/SecuraChain/internal/config"
// 	"github.com/pierreleocadie/SecuraChain/internal/core/block"
// 	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
// 	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
// 	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
// )

// func TestPublishBlockToIPFS(t *testing.T) {
// 	t.Parallel()

// 	// Create a new PebbleTransactionDB instance
// 	database, err := blockchaindb.NewBlockchainDB("blockchain")
// 	if err != nil {
// 		t.Fatalf("Error creating PebbleTransactionDB: %v", err)
// 	}

// 	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
// 	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

// 	/*
// 	* FIRST BLOCK
// 	 */

// 	// Create a genesis block
// 	genesisBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)
// 	genesisBlockBytes, err := genesisBlock.Serialize()
// 	if err != nil {
// 		t.Errorf("Error serializing block: %v", err)
// 	}

// 	// Check if the returned block is not nil
// 	if genesisBlock == nil {
// 		t.Errorf("NewBlock returned nil")
// 	}

// 	// Call the AddBlockToBlockchain function
// 	added, message := blockchaindb.AddBlockToBlockchain(genesisBlock, database)

// 	// Verify the result
// 	if !added {
// 		t.Errorf("Expected block to be added, but it was not added")
// 	}
// 	expectedMessage := "Block addded succesfully to the blockchain"
// 	if message != expectedMessage {
// 		t.Errorf("Expected message: %s, but got: %s", expectedMessage, message)
// 	}

// 	ctx := context.Background()

// 	cfg, err := config.LoadConfig("./config-test.yml")
// 	if err != nil {
// 		t.Errorf("Error loading config file : %v", err)
// 	}

// 	ipfsAPI, nodeIpfs, err := ipfs.SpawnNode(ctx, cfg)
// 	if err != nil {
// 		t.Errorf("Error spawning IPFS node : %v", err)
// 	}

// 	if err := blockchaindb.PublishBlockToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, genesisBlock); err != nil {
// 		t.Errorf("Error publishing block to IPFS : %v", err)
// 	}

// 	blockRegister, err := blockchaindb.ReadBlockDataFromFile("blocksRegistry.json")
// 	if err != nil {
// 		t.Errorf("Error reading block data from file : %v", err)
// 	}

// 	newCidBlock := blockRegister.Blocks[0].BlockCid

// 	// Get the block from IPFS
// 	blockIPFS, err := blockchaindb.GetBlockFromIPFS(ctx, ipfsAPI, newCidBlock)
// 	if err != nil {
// 		t.Errorf("Error getting block from IPFS : %v", err)
// 	}

// 	blockDownloaded, err := blockIPFS.Serialize()
// 	if err != nil {
// 		t.Errorf("Error serializing block : %v", err)
// 	}

// 	if !bytes.Equal(genesisBlockBytes, blockDownloaded) {
// 		t.Errorf("Expected block to be equal, but it was not equal")
// 	}

// 	if err := os.RemoveAll("blockchain"); err != nil {
// 		t.Errorf("Error removing database: %v", err)
// 	}
// 	if err := os.Remove("blocksRegistry.json"); err != nil {
// 		t.Errorf("Error removing database: %v", err)
// 	}
// }
