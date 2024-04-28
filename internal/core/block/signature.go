package block

import (
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// VerifyBlock checks if the block signature is valid
func VerifyBlock(log *ipfsLog.ZapEventLogger, currentBlock *Block) bool {
	if len(currentBlock.Header.Signature) == 0 {
		log.Errorln("Block validation failed: Signature is empty")
		return false
	}

	headerHash := ComputeHash(currentBlock)

	ecdsaPublicKey, err := ecdsa.PublicKeyFromBytes(currentBlock.MinerAddr)
	if err != nil {
		log.Errorln("Block validation failed: %s", err)
		return false
	}

	return ecdsa.VerifySignature(ecdsaPublicKey, headerHash, currentBlock.Header.Signature)
}
