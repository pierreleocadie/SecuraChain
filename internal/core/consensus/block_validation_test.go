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
	stopMiningChan := make(chan StopMiningSignal)
	genesisValidator := DefaultGenesisBlockValidator{}
	txValidatorFactory := DefaultTransactionValidatorFactory{}
	blockValidator := NewDefaultBlockValidator(genesisValidator, txValidatorFactory)

	// Create a previous block
	prevBlock := block.NewBlock(transactions, nil, 1, minerKeyPair)

	MineBlock(prevBlock, stopMiningChan)

	err := prevBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("failed to sign block: %s", err)
	}

	if err := blockValidator.Validate(*prevBlock, block.Block{}); err != nil {
		t.Errorf("validateBlock failed for the genesis block")
	}

	// Create a valid block
	prevBlockHash := block.ComputeHash(*prevBlock)

	validBlock := block.NewBlock(transactions, prevBlockHash, 2, minerKeyPair)

	MineBlock(validBlock, stopMiningChan)

	err = validBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("failed to sign block: %s", err)
	}

	if err := blockValidator.Validate(*validBlock, *prevBlock); err != nil {
		t.Errorf("validateBlock failed for a valid block")
	}

	// Create an invalid block (wrong previous hash)
	invalidBlock := block.NewBlock(transactions, []byte("wronghash"), 3, minerKeyPair)

	MineBlock(invalidBlock, stopMiningChan)

	err = invalidBlock.SignBlock(minerKeyPair)
	if err != nil {
		t.Errorf("failed to sign block: %s", err)
	}

	if err := blockValidator.Validate(*invalidBlock, *validBlock); err == nil {
		t.Errorf("validateBlock succeeded for an invalid block with incorrect previous hash")
	}
}
