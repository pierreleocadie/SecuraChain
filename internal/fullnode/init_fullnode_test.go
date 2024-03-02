package fullnode_test

import (
	"os"
	"testing"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
)

func TestHasABlockchain_BlockchainDoesNotExist(t *testing.T) {
	t.Parallel()

	// Remove the blockchain directory if it exists
	os.RemoveAll("./blockchain")

	// Call the HasABlockchain function
	hasBlockchain := fullnode.HasABlockchain()

	// Verify that the function returns false
	if hasBlockchain {
		t.Errorf("Expected HasABlockchain to return false, got true")
	}
}

func TestHasABlockchain_BlockchainExistsAndIsUpToDate(t *testing.T) {
	t.Parallel()

	// Remove the blockchain directory if it exists to make sure it doesn't exist
	os.RemoveAll("./blockchain")

	// Create a dummy blockchain
	pebbleDB, err := pebble.NewPebbleTransactionDB("./blockchain")
	if err != nil {
		t.Fatalf("Error creating pebble database: %v", err)
	}

	// Set the modification time of the blockchain directory to the current time
	currentTime := time.Now()
	os.Chtimes("./blockchain", currentTime, currentTime)

	// Call the HasABlockchain function
	hasBlockchain := fullnode.HasABlockchain()

	// Verify that the function returns true
	if !hasBlockchain {
		t.Errorf("Expected HasABlockchain to return true, got false")
	}

	pebbleDB.Close()
	os.RemoveAll("./blockchain")
}

func TestHasABlockchain_BlockchainExistsAndIsNotUpToDate(t *testing.T) {

	// Remove the blockchain directory if it exists to make sure it doesn't exist
	os.RemoveAll("./blockchain")

	// Create a dummy blockchain
	pebbleDB, err := pebble.NewPebbleTransactionDB("./blockchain")
	if err != nil {
		t.Fatalf("Error creating pebble database: %v", err)
	}

	// Set the modification time of the blockchain directory to more than 1 hour ago
	lastModifiedTime := time.Now().Add(-2 * time.Hour)
	os.Chtimes("./blockchain", lastModifiedTime, lastModifiedTime)

	// Call the HasABlockchain function
	hasBlockchain := fullnode.HasABlockchain()

	// Verify that the function returns false
	if hasBlockchain {
		t.Errorf("Expected HasABlockchain to return false, got true")
	}

	pebbleDB.Close()
	os.RemoveAll("./blockchain")
}
