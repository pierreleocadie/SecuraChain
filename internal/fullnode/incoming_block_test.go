package fullnode_test

import (
	"os"
	"testing"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func TestProcessBlock_GenesisBlock_Valid(t *testing.T) {
	t.Parallel()

	// Create a new PebbleDB instance
	database, err := pebble.NewBlockchainDB("blockchainDB")
	if err != nil {
		t.Fatalf("Error creating PebbleDB: %v", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	// Create a previous block
	prevBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

	consensus.MineBlock(prevBlock)

	err = prevBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	// Process the genesis block
	processd, err := fullnode.ProcessBlock(prevBlock, database)
	if err != nil {
		t.Fatalf("Error processing genesis block: %v", err)
	}

	// Verify the result
	if !processd {
		t.Errorf("Expected genesis block to be added, but it was not added")
	}

	if err = os.RemoveAll("blockchainDB"); err != nil {
		t.Fatalf("Error removing blockchainDB: %v", err)
	}
}

func TestProcessBlock_GenesisBlock_Invalid(t *testing.T) {
	t.Parallel()

	// Create a new PebbleDB instance
	database, err := pebble.NewBlockchainDB("blockchainDB")
	if err != nil {
		t.Fatalf("Error creating PebbleDB: %v", err)
	}
	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	// Create an invalid genesis block
	genesisBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

	// if we uncomment that the block become valid consensus.MineBlock(genesisBlock)

	err = genesisBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	// Process the genesis block
	processd, err := fullnode.ProcessBlock(genesisBlock, database)
	if err != nil {
		if err.Error() != "genesis block is invalid" {
			t.Errorf("Expected error message: genesis block is invalid, but got: %s", err.Error())
		}
	}

	// Verify the result
	if processd {
		t.Errorf("Expected genesis block to not be added, but it was added")
	}

	if err = os.RemoveAll("blockchainDB"); err != nil {
		t.Fatalf("Error removing blockchainDB: %v", err)
	}

}

func TestProcessBlock_NonGenesisBlock_Valid(t *testing.T) {
	t.Parallel()

	// Create a new PebbleDB instance
	database, err := pebble.NewBlockchainDB("blockchainDB")
	if err != nil {
		t.Fatalf("Error creating PebbleDB: %v", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	/*
	* PREVIOUS BLOCK
	 */

	// Create a previous block
	prevBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

	consensus.MineBlock(prevBlock)

	err = prevBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	// Generate a key for the previous block
	key := block.ComputeHash(prevBlock)

	// Check if the returned block is not nil
	if prevBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Create a second block
	secondBlock := block.NewBlock(transactions, key, 2, minerKeyPair)

	consensus.MineBlock(secondBlock)

	err = secondBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	// Check if the returned block is not nil
	if secondBlock == nil {
		t.Errorf("NewBlock returned nil")
	}
	// Process the genesis block
	processd, err := fullnode.ProcessBlock(prevBlock, database)
	if err != nil {
		t.Fatalf("Error processing genesis block: %v", err)
	}

	// Verify the result
	if !processd {
		t.Errorf("Expected genesis block to be added, but it was not added")
	}

	// Process the non-genesis block
	added, err := fullnode.ProcessBlock(secondBlock, database)
	if err != nil {
		t.Fatalf("Error processing non-genesis block: %v", err)
	}

	// Verify the result
	if !added {
		t.Errorf("Expected non-genesis block to be added, but it was not added")
	}

	if err = os.RemoveAll("blockchainDB"); err != nil {
		t.Fatalf("Error removing blockchainDB: %v", err)
	}
}

func TestProcessBlock_NonGenesisBlock_Invalid(t *testing.T) {
	t.Parallel()

	// Create a new PebbleDB instance
	database, err := pebble.NewBlockchainDB("blockchainDB")
	if err != nil {
		t.Fatalf("Error creating PebbleDB: %v", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	/*
	* PREVIOUS BLOCK
	 */

	// Create a previous block
	prevBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

	consensus.MineBlock(prevBlock)

	err = prevBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	// Generate a key for the previous block
	key := block.ComputeHash(prevBlock)

	// Check if the returned block is not nil
	if prevBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Create a second block
	secondBlock := block.NewBlock(transactions, key, 2, minerKeyPair)

	// we comment that to have a second block invalid consensus.MineBlock(secondBlock)

	err = secondBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	// Check if the returned block is not nil
	if secondBlock == nil {
		t.Errorf("NewBlock returned nil")
	}
	// Process the genesis block
	processd, err := fullnode.ProcessBlock(prevBlock, database)
	if err != nil {
		t.Fatalf("Error processing genesis block: %v", err)
	}

	// Verify the result
	if !processd {
		t.Errorf("Expected genesis block to be added, but it was not added")
	}

	// Process the genesis block
	processdSecondBlock, err := fullnode.ProcessBlock(secondBlock, database)
	if err != nil {
		if err.Error() != "block is invalid" {
			t.Errorf("Expected error message: block is invalid, but got: %s", err.Error())
		}
	}

	// Verify the result
	if processdSecondBlock {
		t.Errorf("Expected block to not be added, but it was added")
	}

	if err = os.RemoveAll("blockchainDB"); err != nil {
		t.Fatalf("Error removing blockchainDB: %v", err)
	}
}
func TestPrevBlockStored_BlockStored(t *testing.T) {
	t.Parallel()

	// Create a new PebbleDB instance
	database, err := pebble.NewBlockchainDB("blockchainDB")
	if err != nil {
		t.Fatalf("Error creating PebbleDB: %v", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	/*
	* PREVIOUS BLOCK
	 */

	// Create a previous block
	prevBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

	consensus.MineBlock(prevBlock)

	err = prevBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	// Generate a key for the previous block
	key := block.ComputeHash(prevBlock)

	// Check if the returned block is not nil
	if prevBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	if err = database.SaveBlock(key, prevBlock); err != nil {
		t.Fatalf("Error saving block: %v", err)
	}

	// Create a second block
	secondBlock := block.NewBlock(transactions, key, 2, minerKeyPair)

	consensus.MineBlock(secondBlock)

	err = secondBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	// Check if the returned block is not nil
	if secondBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	stored, err := fullnode.PrevBlockStored(secondBlock, database)
	if err != nil {
		t.Fatalf("Error checking if the previous block is stored: %v", err)
	}

	// Verify the result
	if !stored {
		t.Errorf("Expected previous block to be stored, but it was not stored")
	}

	if err = os.RemoveAll("blockchainDB"); err != nil {
		t.Fatalf("Error removing blockchainDB: %v", err)
	}
}

func TestPrevBlockStored_BlockNotStored(t *testing.T) {
	t.Parallel()

	// Create a new PebbleDB instance
	database, err := pebble.NewBlockchainDB("blockchainDB")
	if err != nil {
		t.Fatalf("Error creating PebbleDB: %v", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	/*
	* PREVIOUS BLOCK
	 */

	// Create a previous block
	prevBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

	consensus.MineBlock(prevBlock)

	err = prevBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	// Generate a key for the previous block
	key := block.ComputeHash(prevBlock)

	// Check if the returned block is not nil
	if prevBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Create a second block
	secondBlock := block.NewBlock(transactions, key, 2, minerKeyPair)

	consensus.MineBlock(secondBlock)

	err = secondBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	// Check if the returned block is not nil
	if secondBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	stored, err := fullnode.PrevBlockStored(secondBlock, database)
	if err != nil {
		if err.Error() != "previous block is not stored" {
			t.Errorf("Expected error message: previous block not found, but got: %s", err.Error())
		}
	}

	// Verify the result
	if stored {
		t.Errorf("Expected previous block to not be stored, but it was stored")
	}

	if err = os.RemoveAll("blockchainDB"); err != nil {
		t.Fatalf("Error removing blockchainDB: %v", err)
	}
}

func TestCompareBlocksToBlockchain(t *testing.T) {
	t.Parallel()

	// Create a new PebbleDB instance
	database, err := pebble.NewBlockchainDB("blockchainDB")
	if err != nil {
		t.Fatalf("Error creating PebbleDB: %v", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	/*
	* PREVIOUS BLOCK
	 */

	// Create a previous block
	prevBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

	consensus.MineBlock(prevBlock)

	// Generate a key for the previous block
	key := block.ComputeHash(prevBlock)

	err = prevBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	// Save the block the previous block in the database
	err = database.SaveBlock(key, prevBlock)
	if err != nil {
		t.Errorf("Error saving block: %v", err)
	}

	/*
	* SECOND BLOCK
	 */

	// Create a second block
	secondBlock := block.NewBlock(transactions, key, 2, minerKeyPair)

	consensus.MineBlock(secondBlock)

	err = secondBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	blockBuffer := make(map[int64]*block.Block)

	// Add the blocks to the buffer
	blockBuffer[prevBlock.Timestamp] = prevBlock
	blockBuffer[secondBlock.Timestamp] = secondBlock

	// Call the function under test
	blockSorted := fullnode.CompareBlocksToBlockchain(blockBuffer, database)

	// Verify the result
	if len(blockSorted) != 1 {
		t.Errorf("Expected block buffer to contain 1 block, but got: %d", len(blockSorted))
	}

	if err = os.RemoveAll("blockchainDB"); err != nil {
		t.Fatalf("Error removing blockchainDB: %v", err)
	}
}

// func TestHandleIncomingBlock(t *testing.T) {
// 	t.Parallel()

// 	// Create a new PebbleDB instance
// 	database, err := pebble.NewBlockchainDB("blockchainDB")
// 	if err != nil {
// 		t.Fatalf("Error creating PebbleDB: %v", err)
// 	}

// 	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
// 	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

// 	/*
// 	* PREVIOUS BLOCK
// 	 */

// 	// Create a previous block
// 	prevBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

// 	consensus.MineBlock(prevBlock)

// 	// Generate a key for the previous block
// 	key := block.ComputeHash(prevBlock)

// 	err = prevBlock.SignBlock(minerKeyPair)
// 	if err != nil {
// 		t.Errorf("Failed to sign block: %s", err)
// 	}

// 	/*
// 	* SECOND BLOCK
// 	 */

// 	// Create a second block
// 	secondBlock := block.NewBlock(transactions, key, 2, minerKeyPair)

// 	consensus.MineBlock(secondBlock)

// 	err = secondBlock.SignBlock(minerKeyPair)
// 	if err != nil {
// 		t.Errorf("Failed to sign block: %s", err)
// 	}

// 	// Create an incoming block
// 	incomingBlock := block.NewBlock(nil, nil, 1, nil)

// 	blockBuffer := make(map[int64]*block.Block)

// 	// Call the HandleIncomingBlock function
// 	_, err = HandleIncomingBlock(incomingBlock, blockBuffer, database)
// 	if err != nil {
// 		t.Errorf("Error handling incoming block: %v", err)
// 	}

// 	// Verify the result

// 	// Add your assertions here
// }
