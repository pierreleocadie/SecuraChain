package consensus

import (
	"log"

	"github.com/pierreleocadie/SecuraChain/internal/core"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// VerifyBlockSignature checks if the block signature is valid
func VerifyBlockSignature(block *core.Block) bool {
	if len(block.Header.Signature) == 0 {
		log.Printf("Block validation failed: Signature is empty")
		return false
	}

	headerHash := core.ComputeHash(block)

	ecdsaPublicKey, err := ecdsa.PublicKeyFromBytes(block.MinerAddr)
	if err != nil {
		log.Printf("Block validation failed: %s", err)
		return false
	}

	return ecdsa.VerifySignature(ecdsaPublicKey, headerHash[:], block.Header.Signature)
}
