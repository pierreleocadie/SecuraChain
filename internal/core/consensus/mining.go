package consensus

import (
	"math"
	"math/big"

	"github.com/pierreleocadie/SecuraChain/internal/core"
)

// MineBlock performs the mining operation for a new block
func MineBlock(block *core.Block) {
	var maxNonce uint32 = math.MaxUint32

	target := big.NewInt(1)
	target.Lsh(target, uint(256-block.Header.TargetBits))

	for nonce := uint32(0); nonce < maxNonce; nonce++ {
		block.Header.Nonce = nonce
		hash := core.ComputeHash(block)
		hashInt := new(big.Int)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(target) == -1 {
			block.Header.Nonce = nonce
			break
		}
	}
}
