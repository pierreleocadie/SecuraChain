package ecdsa

import (
	"encoding/pem"
	"fmt"
	"os"
	"path"

	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

const (
	filePerm        = 0600
	pemTypeECDSAKey = "ECDSA KEY"
	fileExtPEM      = ".pem"
)

// SaveKeys saves the private and public keys of the ECDSA key pair to specified files.
// The keys are saved in PEM format.
func (keyPair *ecdsaKeyPair) SaveKeys(privateKeyFilename, publicKeyFilename, storagePath string) error {
	if err := keyPair.saveKeyToFile(keyPair.PrivateKeyToBytes, privateKeyFilename, storagePath); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	if err := keyPair.saveKeyToFile(keyPair.PublicKeyToBytes, publicKeyFilename, storagePath); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	return nil
}

// saveKeyToFile writes the provided key to a file in PEM format.
// It constructs the full path using the storagePath and filename parameters.
func (keyPair *ecdsaKeyPair) saveKeyToFile(keyFunc func() ([]byte, error), filename, storagePath string) error {
	keyBytes, err := keyFunc()
	if err != nil {
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

	return os.WriteFile(filePath, pemBytes, filePerm)
}

// LoadKeys retrieves the private and public keys from specified files and constructs an ECDSA key pair.
func LoadKeys(privateKeyFilename, publicKeyFilename, storagePath string) (KeyPair, error) {
	privateKeyBytes, err := utils.LoadKeyFromFile(privateKeyFilename, storagePath)
	if err != nil {
		return nil, err
	}

	privateKey, err := PrivateKeyFromBytes(privateKeyBytes)
	if err != nil {
		return nil, err
	}

	publicKeyBytes, err := utils.LoadKeyFromFile(publicKeyFilename, storagePath)
	if err != nil {
		return nil, err
	}

	publicKey, err := PublicKeyFromBytes(publicKeyBytes)
	if err != nil {
		return nil, err
	}

	return &ecdsaKeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}
