package consensus

import (
	"math"
	"math/big"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// MineBlock performs the mining operation for a new block
func MineBlock(currentBlock *block.Block) {
	var maxNonce uint32 = math.MaxUint32

	target := big.NewInt(1)
	target.Lsh(target, uint(256-currentBlock.Header.TargetBits))

	for nonce := uint32(0); nonce < maxNonce; nonce++ {
		currentBlock.Header.Nonce = nonce
		hash := block.ComputeHash(currentBlock)
		hashInt := new(big.Int)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(target) == -1 {
			currentBlock.Header.Nonce = nonce
			break
		}
	}
}
