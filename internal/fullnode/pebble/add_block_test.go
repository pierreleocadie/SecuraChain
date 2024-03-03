package pebble_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// TestAddBlockToBlockchain is a unit test function that tests the functionality of the AddBlockToBlockchain function.
// It creates a new PebbleTransactionDB instance, creates a block, and adds the block to the blockchain.
// It verifies that the block is added successfully and checks the returned message.
func TestAddBlockToBlockchain(t *testing.T) {
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

	// Verify that the block is in the database
	key := block.ComputeHash(genesisBlock)

	retrievedBlock, err := database.GetBlock(key)
	if err != nil {
		t.Errorf("Expected block to be in the database, but it was not found: %s", err)
	}

	// Serialize the retrieved block
	retrievedBlockBytes, err := retrievedBlock.Serialize()
	if err != nil {
		t.Errorf("Error serializing the retrieved block: %v", err)
	}

	// Serialize the original block
	originalBlockBytes, err := genesisBlock.Serialize()
	if err != nil {
		t.Errorf("Error serializing the original block: %v", err)
	}

	// Compare the original block with the retrieved block
	if !bytes.Equal(originalBlockBytes, retrievedBlockBytes) {
		t.Errorf("Retrieved block does not match the original block")
	}

	database.Close()
	os.RemoveAll("blockchain")
}

func TestAddBlockToBlockchain_BlockAlreadyStored(t *testing.T) {
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
	genesisBlock := block.NewBlock(transactions, nil, 0, minerKeyPair)

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

	// Call the AddBlockToBlockchain function again
	added, message = pebble.AddBlockToBlockchain(genesisBlock, database)

	// Verify the result
	if added {
		t.Errorf("Expected block to not be added, but it was added")
	}
	expectedMessage = "Block already exists in the blockchain"

	if message != expectedMessage {
		t.Errorf("Expected message: %s, but got: %s", expectedMessage, message)
	}

	database.Close()
	os.RemoveAll("blockchain")
}
