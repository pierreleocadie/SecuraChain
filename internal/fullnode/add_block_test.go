package fullnode_test

import (
	"fmt"
	"testing"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// TestAddBlockToBlockchain is a unit test function that tests the functionality of the AddBlockToBlockchain function.
// It creates a new PebbleTransactionDB instance, creates a block, and adds the block to the blockchain.
// It verifies that the block is added successfully and checks the returned message.
func TestAddBlockToBlockchain(t *testing.T) {
	t.Parallel()

	// Create a new PebbleTransactionDB instance
	database, err := pebble.NewPebbleTransactionDB("blockchain_test")
	if err != nil {
		t.Fatalf("Error creating PebbleTransactionDB: %v", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	// Create a block
	b := block.NewBlock(transactions, nil, 1, minerKeyPair)

	// Check if the returned block is not nil
	if b == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Call the AddBlockToBlockchain function
	added, message := fullnode.AddBlockToBlockchain(database, b)

	// Verify the result
	if !added {
		t.Errorf("Expected block to be added, but it was not added")
	}
	expectedMessage := fmt.Sprintf("Block added successfully to the blockchain")
	if message != expectedMessage {
		t.Errorf("Expected message: %s, but got: %s", expectedMessage, message)
	}
}

func TestAddBlockToBlockchain_BlockNotExists(t *testing.T) {
	t.Parallel()

	// Create a new PebbleTransactionDB instance
	database, err := pebble.NewPebbleTransactionDB("blockchain_test2")
	if err != nil {
		t.Fatalf("Error creating PebbleTransactionDB: %v", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	// Create a block
	b := block.NewBlock(transactions, nil, 1, minerKeyPair)

	// Check if the returned block is not nil
	if b == nil {
		t.Errorf("NewBlock returned nil")
	}

	key := block.ComputeHash(b)

	// save the block to the database
	err = database.SaveBlock(key, b)

	// Call the AddBlockToBlockchain function
	added, message := fullnode.AddBlockToBlockchain(database, b)

	// Verify the result
	if added {
		t.Errorf("Expected block to not be added, but it was added")
	}
	expectedMessage := fmt.Sprintf("Error checking for the block existence in the database or Block already exists in the blockchain: %s", err)
	if message != expectedMessage {
		t.Errorf("Expected message: %s, but got: %s", expectedMessage, message)
	}
}
