package fullnode_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ipfs/boxo/path"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func TestPublishBlockchainToIPFS(t *testing.T) {
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
	genesisBlock := block.NewBlock(transactions, nil, 0, minerKeyPair)

	// Check if the returned block is not nil
	if genesisBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Call the AddBlockToBlockchain function
	added, message := pebble.AddBlockToBlockchain(database, genesisBlock)

	// Verify the result
	if !added {
		t.Errorf("Expected block to be added, but it was not added")
	}
	expectedMessage := "Block addded succesfully to the blockchain"
	if message != expectedMessage {
		t.Errorf("Expected message: %s, but got: %s", expectedMessage, message)
	}

	ctx := context.Background()

	oldCid := path.ImmutablePath{}

	cfg, err := config.LoadConfig("./config-test.yml")
	if err != nil {
		t.Errorf("Error loading config file : %v", err)
		os.Exit(1)
	}

	ipfsAPI, nodeIpfs, err := ipfs.SpawnNode(ctx, cfg)
	if err != nil {
		t.Errorf("Error spawning IPFS node : %v", err)
		os.Exit(1)
	}

	newCidBlockChain, err := fullnode.PublishBlockchainToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, oldCid)
	if err != nil {
		t.Errorf("Error publishing blockchain to IPFS : %v", err)
		os.Exit(1)
	}

	// Create a temporary directory
	if err = os.MkdirAll("./test", os.ModePerm); err != nil {
		t.Errorf("could not create output dir (%v)", err)
	}

	// Change directory
	os.Chdir("./test")

	// Get the directory from IPFS
	if err = ipfs.GetDirectory(ctx, cfg, ipfsAPI, newCidBlockChain); err != nil {
		t.Errorf("Error getting blockchain from IPFS : %v", err)
	}

	fmt.Printf("Got directory back from IPFS (IPFS path: %s) and wrote it to %s\n", newCidBlockChain.String(), "./")

}
