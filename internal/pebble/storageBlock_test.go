// Test: TestSaveBlock
package pebble_test

import (
	"bytes"
	"testing"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/pebble"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// TestSaveBlock is a unit test function that tests the SaveBlock method of the PebbleTransactionDB struct.
// It creates a new PebbleTransactionDB instance, generates a key pair, creates a previous block, saves the block to the database,
// retrieves the saved block, serializes both the original and retrieved blocks, and compares them to ensure consistency.
func TestSaveAndGetBlock(t *testing.T) {
	t.Parallel()

	// Create a new PebbleTransactionDB instance
	blockchainTest, err := pebble.NewBlockchainDB("blockchain_test")
	if err != nil {
		t.Errorf("Error creating PebbleTransactionDB: %v", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	// Create a previous block
	prevBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

	// Check if the returned block is not nil
	if prevBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Generate a key for the block
	key := block.ComputeHash(prevBlock)

	// Save the block
	err = blockchainTest.SaveBlock(key, prevBlock)
	if err != nil {
		t.Errorf("Error saving block: %v", err)
	}

	// Retrieve the saved block from the database
	retrievedBlock, err := blockchainTest.GetBlock(key)
	if err != nil {
		t.Errorf("Error retrieving block from the database: %v", err)
	}

	// Serialize the original block
	originalBlockBytes, err := prevBlock.Serialize()
	if err != nil {
		t.Errorf("Error serializing the original block: %v", err)
	}

	// Serialize the retrieved block
	retrievedBlockBytes, err := retrievedBlock.Serialize()
	if err != nil {
		t.Errorf("Error serializing the retrieved block: %v", err)
	}

	// Compare the original block with the retrieved block
	if !bytes.Equal(originalBlockBytes, retrievedBlockBytes) {
		t.Errorf("Retrieved block does not match the original block")
	}

}

// Test: TestGetLastBlock
// TestGetLastBlock is a unit test function that tests the GetLastBlock function of the PebbleTransactionDB type.
// It creates a new PebbleTransactionDB instance, generates a key pair, and creates multiple blocks.
// It then retrieves the last block from the database and compares it with the expected block.
// If there is any error during the test, it fails the test with an error message.
func TestGetLastBlock(t *testing.T) {
	t.Parallel()

	// Create a new PebbleTransactionDB instance
	blockchainTest2, err := pebble.NewBlockchainDB("blockchain_test2")
	if err != nil {
		t.Errorf("Error creating PebbleTransactionDB: %v", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	/*
	* PREVIOUS BLOCK
	 */

	// Create a previous block
	prevBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

	// Generate a key for the previous block
	key := block.ComputeHash(prevBlock)

	// Check if the returned block is not nil
	if prevBlock == nil {
		t.Errorf("NewBlock returned nil")
	}
	// Save the block the previous block in the database
	err = blockchainTest2.SaveBlock(key, prevBlock)
	if err != nil {
		t.Errorf("Error saving block: %v", err)
	}

	/*
	* SECOND BLOCK
	 */

	// Create a second block
	secondBlock := block.NewBlock(transactions, key, 2, minerKeyPair)

	// Generate a key for the second blockblock
	keySecondBlock := block.ComputeHash(secondBlock)

	// Check if the returned block is not nil
	if secondBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Save the second block in the database
	err = blockchainTest2.SaveBlock(keySecondBlock, secondBlock)
	if err != nil {
		t.Errorf("Error saving block: %v", err)
	}

	/*
	* THIRD BLOCK
	 */

	// Create a thrid block
	thirdBlock := block.NewBlock(transactions, keySecondBlock, 3, minerKeyPair)

	// Generate a key for the third block
	keyThirdBlock := block.ComputeHash(thirdBlock)

	// Check if the returned block is not nil
	if thirdBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Save the third block in the database
	err = blockchainTest2.SaveBlock(keyThirdBlock, thirdBlock)
	if err != nil {
		t.Errorf("Error saving block: %v", err)
	}

	/*
	* TESTING
	 */

	// Retrieve the last block from the database
	retrievedLastBlock := blockchainTest2.GetLastBlock()

	// Serialize the retrieved last block
	retrievedLastBlockBytes, err := retrievedLastBlock.Serialize()
	if err != nil {
		t.Errorf("Error serializing the retrieved block: %v", err)
	}

	// Serialize the third block
	thirdBlockBytes, err := thirdBlock.Serialize()
	if err != nil {
		t.Errorf("Error serializing the third: %v", err)
	}

	// Compare the third block with the retrieved last block
	if !bytes.Equal(thirdBlockBytes, retrievedLastBlockBytes) {
		t.Errorf("Retrieved lastblock does not match the third block")
	}

	/*
	* TESTING (this one should return an error)
	 */
	// // Try to compare with the second block
	// secondBlockBytes, err := secondBlock.Serialize()
	// if err != nil {
	// 	t.Errorf("Error serializing the second block: %v", err)
	// }
	// // Compare the second block with the retrieved last block
	// if !bytes.Equal(secondBlockBytes, retrievedLastBlockBytes) {
	// 	t.Errorf("Retrieved lastblock does not with match the second block")
	// }

}

// Test: TestVerifyBlockchainIntegrity
func TestVerifyBlockchainIntegrity(t *testing.T) {
	t.Parallel()

	// Create a new PebbleTransactionDB instance
	blockchainTestV, err := pebble.NewBlockchainDB("blockchain_test_verify_integrity")
	if err != nil {
		t.Errorf("Error creating PebbleTransactionDB: %v", err)
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

	// Check if the returned block is not nil
	if prevBlock == nil {
		t.Errorf("NewBlock returned nil")
	}
	// Save the block the previous block in the database
	err = blockchainTestV.SaveBlock(key, prevBlock)
	if err != nil {
		t.Errorf("Error saving block: %v", err)
	}

	/*
	* SECOND BLOCK
	 */

	// Create a second block
	secondBlock := block.NewBlock(transactions, key, 2, minerKeyPair)

	consensus.MineBlock(secondBlock)

	// Generate a key for the second blockblock
	keySecondBlock := block.ComputeHash(secondBlock)

	// Check if the returned block is not nil
	if secondBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Save the second block in the database
	err = blockchainTestV.SaveBlock(keySecondBlock, secondBlock)
	if err != nil {
		t.Errorf("Error saving block: %v", err)
	}

	/*
	* THIRD BLOCK
	 */

	// Create a thrid block
	thirdBlock := block.NewBlock(transactions, keySecondBlock, 3, minerKeyPair)

	consensus.MineBlock(thirdBlock)

	// Generate a key for the third block
	keyThirdBlock := block.ComputeHash(thirdBlock)

	// Check if the returned block is not nil
	if thirdBlock == nil {
		t.Errorf("NewBlock returned nil")
	}

	// Save the third block in the database
	err = blockchainTestV.SaveBlock(keyThirdBlock, thirdBlock)
	if err != nil {
		t.Errorf("Error saving block: %v", err)
	}

	// Call the VerifyBlockchainIntegrity function
	valid, err := blockchainTestV.VerifyBlockchainIntegrity(blockchainTestV.GetLastBlock())
	if err != nil {
		t.Errorf("Error verifying blockchain integrity: %v", err)
	}

	// Check if the blockchain integrity is valid
	if !valid {
		t.Errorf("Blockchain integrity verification failed")
	}
}
