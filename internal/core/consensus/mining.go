package consensus

import (
	"math"
	"math/big"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

const (
	sha256bits = 256
)

type StopMiningSignal struct {
	Stop          bool
	BlockReceived *block.Block
}

// MineBlock performs the mining operation for a new block
func MineBlock(currentBlock *block.Block, stop chan StopMiningSignal) (bool, *block.Block) {
	var maxNonce uint32 = math.MaxUint32

	target := big.NewInt(1)
	target.Lsh(target, uint(sha256bits-currentBlock.Header.TargetBits))

	for nonce := uint32(0); nonce < maxNonce; nonce++ {
		select {
		case stopSignal := <-stop:
			// Stop early
			return stopSignal.Stop, stopSignal.BlockReceived
		default:
			currentBlock.Header.Nonce = nonce
			hash := block.ComputeHash(currentBlock)
			hashInt := new(big.Int)
			hashInt.SetBytes(hash)

			if hashInt.Cmp(target) == -1 {
				currentBlock.Header.Nonce = nonce
				// Not stopped early
				return false, nil
			}
		}
	}

	// Not stopped early
	return false, nil
}
