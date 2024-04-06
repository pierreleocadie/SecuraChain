package ecdsa

import (
	"encoding/pem"
	"fmt"
	"os"
	"path"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

const (
	filePerm        = 0600
	pemTypeECDSAKey = "ECDSA KEY"
	fileExtPEM      = ".pem"
)

// SaveKeys saves the private and public keys of the ECDSA key pair to specified files.
// The keys are saved in PEM format.
func (keyPair *ecdsaKeyPair) SaveKeys(log *ipfsLog.ZapEventLogger, privateKeyFilename, publicKeyFilename, storagePath string) error {
	if err := keyPair.saveKeyToFile(log, keyPair.PrivateKeyToBytes, privateKeyFilename, storagePath); err != nil {
		log.Errorln("Failed to save private key: ", err)
		return fmt.Errorf("failed to save private key: %w", err)
	}

	if err := keyPair.saveKeyToFile(log, keyPair.PublicKeyToBytes, publicKeyFilename, storagePath); err != nil {
		log.Errorln("Failed to save public key: ", err)
		return fmt.Errorf("failed to save public key: %w", err)
	}

	log.Debugln("Keys saved successfully")
	return nil
}

// saveKeyToFile writes the provided key to a file in PEM format.
// It constructs the full path using the storagePath and filename parameters.
func (keyPair *ecdsaKeyPair) saveKeyToFile(log *ipfsLog.ZapEventLogger, keyFunc func() ([]byte, error), filename, storagePath string) error {
	keyBytes, err := keyFunc()
	if err != nil {
		log.Errorln("Failed to get key bytes: ", err)
		return err
	}

	// Convert raw key bytes into PEM format
	pemBlock := &pem.Block{
		Type:    pemTypeECDSAKey,
		Headers: nil,
		Bytes:   keyBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	filePath := path.Join(storagePath, filename+fileExtPEM)

	log.Debugln("Saving key to file: ", filePath)
	return os.WriteFile(filePath, pemBytes, filePerm)
}

// LoadKeys retrieves the private and public keys from specified files and constructs an ECDSA key pair.
func LoadKeys(log *ipfsLog.ZapEventLogger, privateKeyFilename, publicKeyFilename, storagePath string) (KeyPair, error) {
	privateKeyBytes, err := utils.LoadKeyFromFile(log, privateKeyFilename, storagePath)
	if err != nil {
		log.Errorln("Failed to load private key: ", err)
		return nil, err
	}

	privateKey, err := PrivateKeyFromBytes(log, privateKeyBytes)
	if err != nil {
		log.Errorln("Failed to create private key: ", err)
		return nil, err
	}

	publicKeyBytes, err := utils.LoadKeyFromFile(log, publicKeyFilename, storagePath)
	if err != nil {
		log.Errorln("Failed to load public key: ", err)
		return nil, err
	}

	publicKey, err := PublicKeyFromBytes(log, publicKeyBytes)
	if err != nil {
		log.Errorln("Failed to create public key: ", err)
		return nil, err
	}

	return &ecdsaKeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}
