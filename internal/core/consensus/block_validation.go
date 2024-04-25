package consensus

import (
	"bytes"
	"math/big"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// ValidateBlock validates the given block
func ValidateBlock(log *ipfsLog.ZapEventLogger, currentBlock *block.Block, prevBlock *block.Block) bool {
	// Special handling for the genesis block
	if currentBlock.Header.Height == 1 && prevBlock == nil {
		return validateGenesisBlock(log, currentBlock)
	}

	// Check if the block's previous hash matches the hash of the previous block
	if currentBlock.Header.Height > 1 && !bytes.Equal(currentBlock.Header.PrevBlock, block.ComputeHash(prevBlock)) {
		log.Warnln("Block validation failed: Previous hash does not match")
		return false
	}

	// Verify the block hash meets the difficulty requirement
	target := big.NewInt(1)
	target.Lsh(target, uint(sha256bits-currentBlock.Header.TargetBits))
	hash := block.ComputeHash(currentBlock)
	hashInt := new(big.Int)
	hashInt.SetBytes(hash)

	if hashInt.Cmp(target) == 1 {
		log.Warnln("Block validation failed: Hash does not meet the target")
		return false
	}

	// Verify the block's timestamp is not too far in the future
	if currentBlock.Header.Timestamp > time.Now().Unix()+10 {
		log.Warnln("Block validation failed: Timestamp is too far in the future")
		return false
	}

	// Verify the block's timestamp is not before the previous block's timestamp
	if currentBlock.Header.Timestamp < prevBlock.Header.Timestamp {
		log.Warnln("Block validation failed: Timestamp is before the previous block's timestamp")
		return false
	}

	// Verify the block's height is one more than the previous block's height
	if currentBlock.Header.Height != prevBlock.Header.Height+1 {
		log.Warnln("Block validation failed: Height is not one more than the previous block's height")
		return false
	}

	// Verify the block's signature is valid
	if !block.VerifyBlock(log, currentBlock) {
		log.Warnln("Block validation failed: Signature is invalid")
		return false
	}

	// If not empty, verify the block's transactions
	if len(currentBlock.Transactions) > 0 {
		for _, tx := range currentBlock.Transactions {
			if !ValidateTransaction(tx) {
				log.Warnln("Block validation failed: Transaction is invalid")
				return false
			}
		}
	}

	return true
}

// ValidateGenesisBlock validates the given genesis block
func validateGenesisBlock(log *ipfsLog.ZapEventLogger, genesisBlock *block.Block) bool {
	// Verify the block hash meets the difficulty requirement
	target := big.NewInt(1)
	target.Lsh(target, uint(sha256bits-genesisBlock.Header.TargetBits))
	hash := block.ComputeHash(genesisBlock)
	hashInt := new(big.Int)
	hashInt.SetBytes(hash)

	if hashInt.Cmp(target) == 1 {
		log.Warnln("Block validation failed: Hash does not meet the target")
		return false
	}

	// Verify the block's timestamp is not too far in the future
	if genesisBlock.Header.Timestamp > time.Now().Unix()+10 {
		log.Warnln("Block validation failed: Timestamp is too far in the future")
		return false
	}

	// Verify the block's height is equal to one (genesis block)
	if genesisBlock.Header.Height != 1 {
		log.Warnln("Block validation failed: Height is not one")
		return false
	}

	// Verify the block's signature is valid
	if !block.VerifyBlock(log, genesisBlock) {
		log.Warnln("Block validation failed: Signature is invalid")
		return false
	}

	return true
}
