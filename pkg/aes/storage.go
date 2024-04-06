package aes

import (
	"encoding/pem"
	"fmt"
	"os"
	"path"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

// Constants for file permissions, PEM type, and file extension.
const (
	filePerm      = 0600      // File permissions for saving the AES key
	pemTypeAESKey = "AES KEY" // PEM type for saving the AES key
	fileExtPEM    = ".pem"    // File extension for the AES key file
)

// SaveKey wraps the internal saveKeyToFile function and provides
// additional error handling. It saves the AES key to a file.
func (aesKey *aesKey) SaveKey(filename, storagePath string) error {
	if err := aesKey.saveKeyToFile(filename, storagePath); err != nil {
		return fmt.Errorf("failed to save key: %w", err)
	}
	return nil
}

// saveKeyToFile saves the AES key in PEM format to a file at the given storage path.
// It returns an error if the file cannot be created or written to.
func (aesKey *aesKey) saveKeyToFile(filename, storagePath string) error {
	// Convert the raw key bytes to PEM format
	pemBlock := &pem.Block{
		Type:    pemTypeAESKey,
		Headers: nil,
		Bytes:   aesKey.key,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Create the full path for the file
	filePath := path.Join(storagePath, filename+fileExtPEM)

	// Write the PEM-formatted key bytes to the file
	return os.WriteFile(filePath, pemBytes, filePerm)
}

// LoadKey reads an AES key from a file and returns it as an AESKey object.
// It returns an error if the key cannot be read or is invalid.
func LoadKey(log *ipfsLog.ZapEventLogger, filename, storagePath string) (Key, error) {
	keyBytes, err := utils.LoadKeyFromFile(log, filename, storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load key: %w", err)
	}

	if len(keyBytes) != KeySize {
		return nil, fmt.Errorf("invalid key size: %d", len(keyBytes))
	}

	return &aesKey{key: keyBytes}, nil
}
