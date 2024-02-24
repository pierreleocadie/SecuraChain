package consensus

import (
	"bytes"
	"log"
	"math/big"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
)

// ValidateBlock validates the given block
func ValidateBlock(currentBlock *block.Block, prevBlock *block.Block) bool {
	// Special handling for the genesis block
	if currentBlock.Header.Height == 1 && prevBlock == nil {
		return validateGenesisBlock(currentBlock)
	}

	// Check if the block's previous hash matches the hash of the previous block
	if currentBlock.Header.Height > 1 && !bytes.Equal(currentBlock.Header.PrevBlock, block.ComputeHash(prevBlock)) {
		log.Printf("Block validation failed: Previous hash does not match")
		return false
	}

	// Verify the block hash meets the difficulty requirement
	target := big.NewInt(1)
	target.Lsh(target, uint(sha256bits-currentBlock.Header.TargetBits))
	hash := block.ComputeHash(currentBlock)
	hashInt := new(big.Int)
	hashInt.SetBytes(hash)

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
	if !block.VerifyBlock(currentBlock) {
		log.Printf("Block validation failed: Signature is invalid")
		return false
	}

	return true
}

// ValidateGenesisBlock validates the given genesis block
func validateGenesisBlock(genesisBlock *block.Block) bool {
	// Verify the block hash meets the difficulty requirement
	target := big.NewInt(1)
	target.Lsh(target, uint(sha256bits-genesisBlock.Header.TargetBits))
	hash := block.ComputeHash(genesisBlock)
	hashInt := new(big.Int)
	hashInt.SetBytes(hash)

	if hashInt.Cmp(target) == 1 {
		log.Printf("Block validation failed: Hash does not meet the target")
		return false
	}

	// Verify the block's timestamp is not too far in the future
	if genesisBlock.Header.Timestamp > time.Now().Unix()+10 {
		log.Printf("Block validation failed: Timestamp is too far in the future")
		return false
	}

	// Verify the block's height is equal to one (genesis block)
	if genesisBlock.Header.Height != 1 {
		log.Printf("Block validation failed: Height is not one")
		return false
	}

	// Verify the block's signature is valid
	if !block.VerifyBlock(genesisBlock) {
		log.Printf("Block validation failed: Signature is invalid")
		return false
	}

	return true
}

// ValidateListBlock validates a list of blocks
func ValidateListBlock(blockList []*block.Block, database *pebble.PebbleTransactionDB) bool {
	for i := 0; i < len(blockList); i++ {
		if i == 0 {
			if !validateGenesisBlock(blockList[i]) {
				return false
			}
		} else {
			if i == 0 {
				prevBlock, err := database.GetBlock(blockList[i].PrevBlock)
				if err != nil {
					return false
				}
				if !ValidateBlock(blockList[i], prevBlock) {
					return false
				}
			}

			if !ValidateBlock(blockList[i], blockList[i-1]) {
				return false
			}
		}
	}
	return true
}
