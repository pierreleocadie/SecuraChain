package consensus

import (
	"math/big"
	"testing"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// TestMineBlock tests the mining of a block
func TestMineBlock(t *testing.T) {
	t.Parallel()

	log := ipfsLog.Logger("mining-test")
	if err := ipfsLog.SetLogLevel("mining-test", "DEBUG"); err != nil {
		log.Errorln("Failed to set log level : ", err)
	}

	// Create a sample block
	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()  // Replace with actual key pair generation
	transactions := []transaction.Transaction{} // Empty transaction list for simplicity

	newBlock := block.NewBlock(log, transactions, []byte("GenesisBlock"), 1, minerKeyPair)

	// Mine the block
	MineBlock(log, newBlock)

	// Verify if the mined block's hash satisfies the target
	target := big.NewInt(1)
	target.Lsh(target, uint(sha256bits-newBlock.Header.TargetBits))

	hash := block.ComputeHash(log, newBlock)
	hashInt := new(big.Int)
	hashInt.SetBytes(hash)

	if hashInt.Cmp(target) >= 0 {
		t.Errorf("Block mining failed, hash does not meet the target. Hash: %x", hash)
	}
}
