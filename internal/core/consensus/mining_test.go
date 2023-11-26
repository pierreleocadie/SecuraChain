package consensus

import (
	"math/big"
	"testing"

	"github.com/pierreleocadie/SecuraChain/internal/core"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// TestMineBlock tests the mining of a block
func TestMineBlock(t *testing.T) {
	t.Parallel()

	// Create a sample block
	minerKeyPair, _ := ecdsa.NewECDSAKeyPair() // Replace with actual key pair generation
	transactions := []core.Transaction{}       // Empty transaction list for simplicity

	t.Logf("Creating block")
	block := core.NewBlock(transactions, []byte("GenesisBlock"), 1, minerKeyPair)

	// Mine the block
	t.Logf("Mining block")
	MineBlock(block)

	// Verify if the mined block's hash satisfies the target
	target := big.NewInt(1)
	target.Lsh(target, uint(256-block.Header.TargetBits))

	t.Logf("Verifying block hash")
	hash := core.ComputeHash(block)
	hashInt := new(big.Int)
	hashInt.SetBytes(hash)

	if hashInt.Cmp(target) >= 0 {
		t.Errorf("Block mining failed, hash does not meet the target. Hash: %x", hash)
	}

	t.Logf("Block mining succeeded")
}
