package client

import (
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func AskFilesList(cfg *config.Config, ecdsaKeyPair ecdsa.KeyPair, askFilesListChan chan []byte, log *ipfsLog.ZapEventLogger) error {
	pubKey, err := ecdsaKeyPair.PublicKeyToBytes()
	if err != nil {
		log.Errorf("failed to get public key: %s", err)
		return err
	}
	askFilesListChan <- pubKey
	return nil
}
