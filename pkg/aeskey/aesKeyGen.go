// Package aeskey provides functionality to generate and manage 32 byte key for AES256.
package aeskey

import "crypto/rand"

// KeySize represents the size in bytes of the AES key (32 bytes for AES256).
const KeySize = 32

// AesKey represents a unique random key on 32 byte.
type AesKey struct {
	key []byte
}

// NewKey generates a new AES key.
func NewKey() *AesKey {
	key := make([]byte, KeySize) // generate a random 32 byte key for AES256.
	if _, err := rand.Read(key); err != nil {
		panic(err.Error())
	}

	return &AesKey{key: key}
}

// String provides a string representation of the AesKey, which is byte string.
func (keygen *AesKey) String() string {
	return string(keygen.key)
}

// GetKey returns the key byte slice of an AesKey.
func (keygen *AesKey) GetKey() []byte {
	return keygen.key
}
