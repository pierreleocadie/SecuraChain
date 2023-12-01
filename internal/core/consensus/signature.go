package consensus

import (
	"log"

	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// VerifyBlockSignature checks if the block signature is valid
func VerifyBlockSignature(currentBlock *block.Block) bool {
	if len(currentBlock.Header.Signature) == 0 {
		log.Printf("Block validation failed: Signature is empty")
		return false
	}

	headerHash := block.ComputeHash(currentBlock)

	ecdsaPublicKey, err := ecdsa.PublicKeyFromBytes(currentBlock.MinerAddr)
	if err != nil {
		log.Printf("Block validation failed: %s", err)
		return false
	}

	return ecdsa.VerifySignature(ecdsaPublicKey, headerHash[:], currentBlock.Header.Signature)
}
