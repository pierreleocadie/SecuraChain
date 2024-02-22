package node

import (
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

func GenerateKeys(cfg *config.Config, log *ipfsLog.ZapEventLogger) {
	ecdsaKeyPairPath, err := utils.SanitizePath(cfg.ECDSAKeyPairPath)
	if err != nil {
		log.Panicf("Failed to sanitize ECDSA key pair path: %s", err)
	}

	ecdsaKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		log.Panicf("Failed to create ECDSA key pair: %s", err)
	}

	err = ecdsaKeyPair.SaveKeys("ecdsaPrivateKey", "ecdsaPublicKey", ecdsaKeyPairPath)
	if err != nil {
		log.Panicf("Failed to save ECDSA key pair: %s", err)
	}

	aesKeyPath, err := utils.SanitizePath(cfg.AESKeyPath)
	if err != nil {
		log.Panicf("Failed to sanitize AES key path: %s", err)
	}

	aesKey, err := aes.NewAESKey()
	if err != nil {
		log.Panicf("Failed to create AES key: %s", err)
	}

	err = aesKey.SaveKey("aesKey", aesKeyPath)
	if err != nil {
		log.Panicf("Failed to save AES key: %s", err)
	}

	log.Info("Keys generated")
}

func LoadKeys(cfg *config.Config, log *ipfsLog.ZapEventLogger) (ecdsa.KeyPair, aes.Key) {
	ecdsaKeyPairPath, err := utils.SanitizePath(cfg.ECDSAKeyPairPath)
	if err != nil {
		log.Panicf("Failed to sanitize ECDSA key pair path: %s", err)
	}

	ecdsaKeyPair, err := ecdsa.LoadKeys("ecdsaPrivateKey", "ecdsaPublicKey", ecdsaKeyPairPath)
	if err != nil {
		log.Panicf("Failed to load ECDSA key pair: %s", err)
	}

	aesKeyPath, err := utils.SanitizePath(cfg.AESKeyPath)
	if err != nil {
		log.Panicf("Failed to sanitize AES key path: %s", err)
	}

	aesKey, err := aes.LoadKey("aesKey", aesKeyPath)
	if err != nil {
		log.Panicf("Failed to load AES key: %s", err)
	}

	log.Info("Keys loaded")
	return ecdsaKeyPair, aesKey
}
