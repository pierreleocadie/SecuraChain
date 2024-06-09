package consensus

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"reflect"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// ValidateBlock validates the given block
func ValidateBlock(currentBlock, prevBlock block.Block) error {
	if reflect.DeepEqual(currentBlock, block.Block{}) {
		return fmt.Errorf("Block validation failed: current block is empty")
	}

	// Special handling for the genesis block
	if currentBlock.Header.Height == 1 && reflect.DeepEqual(prevBlock, block.Block{}) {
		return validateGenesisBlock(currentBlock)
	}

	if reflect.DeepEqual(prevBlock, block.Block{}) {
		return fmt.Errorf("Block validation failed: previous block is empty")
	}

	// Check if the block's previous hash matches the hash of the previous block
	if currentBlock.Header.Height > 1 && !bytes.Equal(currentBlock.Header.PrevBlock, block.ComputeHash(prevBlock)) {
		return fmt.Errorf("Block validation failed: Previous hash does not match")
	}

	// Verify the block hash meets the difficulty requirement
	target := big.NewInt(1)
	target.Lsh(target, uint(sha256bits-currentBlock.Header.TargetBits))
	hash := block.ComputeHash(currentBlock)
	hashInt := new(big.Int)
	hashInt.SetBytes(hash)

	if hashInt.Cmp(target) == 1 {
		return fmt.Errorf("Block validation failed: Hash does not meet the target")
	}

	// Verify the block's timestamp is not too far in the future
	if currentBlock.Header.Timestamp > time.Now().UTC().Unix()+10 {
		return fmt.Errorf("Block validation failed: Timestamp is too far in the future")
	}

	// Verify the block's timestamp is not before the previous block's timestamp
	if currentBlock.Header.Timestamp < prevBlock.Header.Timestamp {
		return fmt.Errorf("Block validation failed: Timestamp is before the previous block's timestamp")
	}

	// Verify the block's signature is valid
	if err := block.VerifyBlock(currentBlock); err != nil {
		return fmt.Errorf("Block validation failed: Signature is invalid: %w", err)
	}

	// If not empty, verify the block's transactions
	if len(currentBlock.Transactions) > 0 {
		for _, tx := range currentBlock.Transactions {
			if err := ValidateTransaction(tx); err != nil {
				log.Printf("Block validation failed: Transaction is invalid")
				return fmt.Errorf("Block validation failed: Transaction is invalid")
			}
		}
	}

	return nil
}

// ValidateGenesisBlock validates the given genesis block
func validateGenesisBlock(genesisBlock block.Block) error {
	// Verify the block hash meets the difficulty requirement
	target := big.NewInt(1)
	target.Lsh(target, uint(sha256bits-genesisBlock.Header.TargetBits))
	hash := block.ComputeHash(genesisBlock)
	hashInt := new(big.Int)
	hashInt.SetBytes(hash)

	if hashInt.Cmp(target) == 1 {
		log.Printf("Block validation failed: Hash does not meet the target")
		return fmt.Errorf("Block validation failed: Hash does not meet the target")
	}

	// Verify the block's timestamp is not too far in the future
	if genesisBlock.Header.Timestamp > time.Now().UTC().Unix()+10 {
		log.Printf("Block validation failed: Timestamp is too far in the future")
		return fmt.Errorf("Block validation failed: Timestamp is too far in the future")
	}

	// Verify the block's height is equal to one (genesis block)
	if genesisBlock.Header.Height != 1 {
		log.Printf("Block validation failed: Height is not one")
		return fmt.Errorf("Block validation failed: Height is not one")
	}

	// Verify the block's signature is valid
	if err := block.VerifyBlock(genesisBlock); err != nil {
		log.Printf("Block validation failed: Signature is invalid")
		return fmt.Errorf("Block validation failed: Signature is invalid")
	}

	return nil
}
