// Package aes provides functionality to generate and manage 32 byte key for AES256.
package aes

import "crypto/rand"

// KeySize represents the size in bytes of the AES key (32 bytes for AES256).
const keySize = 32

// AesKey represents a unique random key on 32 byte.
type Aes struct {
	key []byte
}

// NewKey generates a new AES key.
func NewKey() *Aes {
	key := make([]byte, KeySize) // generate a random 32 byte key for AES256.
	if _, err := rand.Read(key); err != nil {
		panic(err.Error())
	}

	return &Aes{key: key}
}

// String provides a string representation of the Aes, which is byte string.
func (keygen *Aes) String() string {
	return string(keygen.key)
}

// GetKey returns the key byte slice of an Aes.
func (keygen *Aes) GetKey() []byte {
	return keygen.key
}
