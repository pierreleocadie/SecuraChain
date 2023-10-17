package ecdsakeypair

import (
	"encoding/pem"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	filePerm        = 0600
	pemTypeECDSAKey = "ECDSA KEY"
	fileExtPEM      = ".pem"
)

// SaveKeys saves the private and public keys of the ECDSA key pair to specified files.
// The keys are saved in PEM format.
func (keyPair *ecdsaKeyPair) SaveKeys(privateKeyFilename, publicKeyFilename, storagePath string) error {
	if err := keyPair.saveKeyToFile(keyPair.privateKeyToBytes, privateKeyFilename, storagePath); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	if err := keyPair.saveKeyToFile(keyPair.publicKeyToBytes, publicKeyFilename, storagePath); err != nil {
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
func LoadKeys(privateKeyFilename, publicKeyFilename, storagePath string) (ECDSAKeyPair, error) {
	privateKeyBytes, err := loadKeyFromFile(privateKeyFilename, storagePath)
	if err != nil {
		return nil, err
	}

	privateKey, err := privateKeyFromBytes(privateKeyBytes)
	if err != nil {
		return nil, err
	}

	publicKeyBytes, err := loadKeyFromFile(publicKeyFilename, storagePath)
	if err != nil {
		return nil, err
	}

	publicKey, err := publicKeyFromBytes(publicKeyBytes)
	if err != nil {
		return nil, err
	}

	return &ecdsaKeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// loadKeyFromFile reads a key from a PEM-formatted file and returns its bytes.
func loadKeyFromFile(filename, storagePath string) ([]byte, error) {
	// Construct and clean the path
	filePath := path.Join(storagePath, filename+fileExtPEM)
	cleanFilePath := filepath.Clean(filePath)

	// Ensure that the cleaned path still has the intended base directory
	if !strings.HasPrefix(cleanFilePath, storagePath) {
		return nil, fmt.Errorf("potential path traversal attempt detected")
	}

	fileBytes, err := os.ReadFile(cleanFilePath)
	if err != nil {
		return nil, err
	}

	// Decode PEM file to get the raw bytes
	block, extraBytes := pem.Decode(fileBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", filename)
	}
	if len(extraBytes) > 0 {
		return nil, fmt.Errorf("unexpected extra bytes in PEM file %s", filename)
	}

	return block.Bytes, nil
}
