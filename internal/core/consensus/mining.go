package consensus

import (
	"math"
	"math/big"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

const (
	sha256bits = 256
)

// MineBlock performs the mining operation for a new block
func MineBlock(log *ipfsLog.ZapEventLogger, currentBlock *block.Block) {
	var maxNonce uint32 = math.MaxUint32

	target := big.NewInt(1)
	target.Lsh(target, uint(sha256bits-currentBlock.Header.TargetBits))

	for nonce := uint32(0); nonce < maxNonce; nonce++ {
		currentBlock.Header.Nonce = nonce
		hash := block.ComputeHash(log, currentBlock)
		hashInt := new(big.Int)
		hashInt.SetBytes(hash)

		if hashInt.Cmp(target) == -1 {
			currentBlock.Header.Nonce = nonce
			break
		}
	}
}
