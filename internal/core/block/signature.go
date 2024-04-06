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

	headerHash := ComputeHash(log, currentBlock)

	ecdsaPublicKey, err := ecdsa.PublicKeyFromBytes(log, currentBlock.MinerAddr)
	if err != nil {
		log.Errorln("Block validation failed: %s", err)
		return false
	}

	log.Debugln("Verifying block signature")
	return ecdsa.VerifySignature(log, ecdsaPublicKey, headerHash, currentBlock.Header.Signature)
}
