package consensus

import (
	"bytes"
	"log"
	"math/big"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// ValidateBlock validates the given block
func ValidateBlock(currentBlock *block.Block, prevBlock *block.Block) bool {
	// Check if the block's previous hash matches the hash of the previous block
	if currentBlock.Header.Height > 0 && !bytes.Equal(currentBlock.Header.PrevBlock, block.ComputeHash(prevBlock)) {
		log.Printf("Block validation failed: Previous hash does not match")
		return false
	}

	// Verify the block hash meets the difficulty requirement
	target := big.NewInt(1)
	target.Lsh(target, uint(256-currentBlock.Header.TargetBits))
	hash := block.ComputeHash(currentBlock)
	hashInt := new(big.Int)
	hashInt.SetBytes(hash[:])

	if hashInt.Cmp(target) == 1 {
		log.Printf("Block validation failed: Hash does not meet the target")
		return false
	}

	// Verify the block's timestamp is not too far in the future
	if currentBlock.Header.Timestamp > time.Now().Unix()+10 {
		log.Printf("Block validation failed: Timestamp is too far in the future")
		return false
	}

	// Verify the block's timestamp is not before the previous block's timestamp
	if currentBlock.Header.Timestamp < prevBlock.Header.Timestamp {
		log.Printf("Block validation failed: Timestamp is before the previous block's timestamp")
		return false
	}

	// Verify the block's height is one more than the previous block's height
	if currentBlock.Header.Height != prevBlock.Header.Height+1 {
		log.Printf("Block validation failed: Height is not one more than the previous block's height")
		return false
	}

	// Verify the block's signature is valid
	if !VerifyBlockSignature(currentBlock) {
		log.Printf("Block validation failed: Signature is invalid")
		return false
	}

	return true
}
