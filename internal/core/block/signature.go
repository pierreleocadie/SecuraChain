package block

import (
	"fmt"
	"log"

	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// VerifyBlock checks if the block signature is valid
func VerifyBlock(currentBlock Block) error {
	if len(currentBlock.Header.Signature) == 0 {
		return fmt.Errorf("Block validation failed: Signature is empty")
	}

	headerHash := ComputeHash(currentBlock)

	ecdsaPublicKey, err := ecdsa.PublicKeyFromBytes(currentBlock.MinerAddr)
	if err != nil {
		log.Printf("Block validation failed: %s", err)
		return fmt.Errorf("Block validation failed: Invalid ECDSA public key: %w", err)
	}

	if !ecdsa.VerifySignature(ecdsaPublicKey, headerHash, currentBlock.Header.Signature) {
		return fmt.Errorf("Block validation failed: Signature is invalid")
	}

	return nil
}
