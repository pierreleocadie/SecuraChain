// Package aes provides functionality for generating and managing
// 32-byte keys suitable for AES-256 encryption.
package aes

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// KeySize defines the size in bytes of the AES key.
// For AES-256, the key size is 32 bytes.
const KeySize = 32

// AESKey defines the interface for an AES key management object.
// It provides methods for converting the key to a string,
// retrieving the key, saving the key to a file, and encrypting
// and decrypting files.
type Key interface {
	String() string
	Key() []byte
	SaveKey(filename, storagePath string) error
	EncryptFile(inputFilePath, outputFilePath string) error
	DecryptFile(inputFilePath, outputFilePath string) error
}

// aesKey is a concrete implementation of AESKey.
// It holds a 32-byte key.
type aesKey struct {
	key []byte
}

// NewAESKey generates a new 32-byte AES key.
// It returns the AESKey object and any error encountered.
func NewAESKey() (Key, error) {
	key := make([]byte, KeySize)
	// Generate a random 32-byte key for AES-256.
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}
	return &aesKey{
		key: key,
	}, nil
}

// String returns a base64 encoded string representation of the AES key.
func (aesKey *aesKey) String() string {
	return base64.StdEncoding.EncodeToString(aesKey.key)
}

// Key returns the byte slice representation of the AES key.
func (aesKey *aesKey) Key() []byte {
	return aesKey.key
}
