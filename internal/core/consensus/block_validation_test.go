// block_validation_test.go
package consensus

import (
	"testing"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func TestValidateBlock(t *testing.T) {
	t.Parallel()

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	// Create a previous block
	t.Logf("Creating previous block")
	prevBlock := block.NewBlock(transactions, []byte("GenesisBlock"), 1, minerKeyPair)
	t.Logf("Mining previous block")
	MineBlock(prevBlock)
	t.Logf("Signing previous block")
	prevBlock.SignBlock(minerKeyPair)

	// Create a valid block
	t.Logf("Computing previous block hash")
	prevBlockHash := block.ComputeHash(prevBlock)

	t.Logf("Creating valid block")
	validBlock := block.NewBlock(transactions, prevBlockHash, 2, minerKeyPair)
	t.Logf("Mining valid block")
	MineBlock(validBlock)
	t.Logf("Signing valid block")
	validBlock.SignBlock(minerKeyPair)

	t.Logf("Validating valid block against previous block")
	if !ValidateBlock(validBlock, prevBlock) {
		t.Errorf("ValidateBlock failed for a valid block")
	}

	// Create an invalid block (wrong previous hash)
	invalidBlock := block.NewBlock(transactions, []byte("wronghash"), 3, minerKeyPair)
	MineBlock(invalidBlock)
	invalidBlock.SignBlock(minerKeyPair)

	t.Logf("Validating invalid block against valid block")
	if ValidateBlock(invalidBlock, validBlock) {
		t.Errorf("ValidateBlock succeeded for an invalid block with incorrect previous hash")
	}
}
