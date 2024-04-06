// block_validation_test.go
package consensus

import (
	"testing"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func TestValidateBlock(t *testing.T) {
	t.Parallel()

	log := ipfsLog.Logger("consenus-test")
	if err := ipfsLog.SetLogLevel("consenus-test", "DEBUG"); err != nil {
		log.Errorln("Failed to set log level : ", err)
	}

	minerKeyPair, _ := ecdsa.NewECDSAKeyPair(log) // Replace with actual key pair generation
	transactions := []transaction.Transaction{}   // Empty transaction list for simplicity

	// Create a previous block
	prevBlock := block.NewBlock(log, transactions, nil, 1, minerKeyPair)

	MineBlock(log, prevBlock)

	err := prevBlock.SignBlock(log, minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	if !ValidateBlock(log, prevBlock, nil) {
		t.Errorf("ValidateBlock failed for the genesis block")
	}

	// Create a valid block
	prevBlockHash := block.ComputeHash(log, prevBlock)

	validBlock := block.NewBlock(log, transactions, prevBlockHash, 2, minerKeyPair)

	MineBlock(log, validBlock)

	err = validBlock.SignBlock(log, minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	if !ValidateBlock(log, validBlock, prevBlock) {
		t.Errorf("ValidateBlock failed for a valid block")
	}

	// Create an invalid block (wrong previous hash)
	invalidBlock := block.NewBlock(log, transactions, []byte("wronghash"), 3, minerKeyPair)

	MineBlock(log, invalidBlock)

	err = invalidBlock.SignBlock(log, minerKeyPair)
	if err != nil {
		t.Errorf("Failed to sign block: %s", err)
	}

	if ValidateBlock(log, invalidBlock, validBlock) {
		t.Errorf("ValidateBlock succeeded for an invalid block with incorrect previous hash")
	}
}
