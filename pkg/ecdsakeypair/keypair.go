// Package ecdsakeypair provides utilities for generating and managing ECDSA key pairs.
package ecdsakeypair

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"fmt"
)

// ECDSAKeyPair represents the interface to interact with an ECDSA key pair.
type ECDSAKeyPair interface {
	PrivateKeyString() (string, error)
	PublicKeyString() (string, error)
	GetPrivateKey() *ecdsa.PrivateKey
	GetPublicKey() *ecdsa.PublicKey
	Sign(hash []byte) ([]byte, error)
	VerifySignature(hash, signature []byte) bool
	SaveKeys(privateKeyFilename, publicKeyFilename, storagePath string) error
}

// ecdsaKeyPair is a struct that encapsulates an ECDSA private and public key pair.
type ecdsaKeyPair struct {
	privateKey *ecdsa.PrivateKey // the private key component of the ECDSA key pair
	publicKey  *ecdsa.PublicKey  // the public key component of the ECDSA key pair
}

// NewECDSAKeyPair initializes a new ECDSA key pair using the P-256 elliptic curve.
// It returns a pointer to an ecdsaKeyPair instance and an error, if any occurred during key generation.
func NewECDSAKeyPair() (ECDSAKeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	return &ecdsaKeyPair{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}, nil
}

// privateKeyToBytes converts the ECDSA private key into a byte slice.
// It returns an error if the conversion fails.
func (keyPair *ecdsaKeyPair) privateKeyToBytes() ([]byte, error) {
	return x509.MarshalECPrivateKey(keyPair.privateKey)
}

// publicKeyToBytes converts the ECDSA public key into a byte slice.
// It returns an error if the conversion fails.
func (keyPair *ecdsaKeyPair) publicKeyToBytes() ([]byte, error) {
	return x509.MarshalPKIXPublicKey(keyPair.publicKey)
}

// PrivateKeyString returns a string representation of the ECDSA private key in hexadecimal format.
// If there's an error during conversion, it returns an error message.
func (keyPair *ecdsaKeyPair) PrivateKeyString() (string, error) {
	bytes, err := keyPair.privateKeyToBytes()
	if err != nil {
		return "", fmt.Errorf("failed to marshal private key to bytes: %s", err)
	}

	return fmt.Sprintf("%x", bytes), nil
}

// PublicKeyString returns a string representation of the ECDSA public key in hexadecimal format.
// If there's an error during conversion, it returns an error message.
func (keyPair *ecdsaKeyPair) PublicKeyString() (string, error) {
	bytes, err := keyPair.publicKeyToBytes()
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key to bytes: %s", err)
	}

	return fmt.Sprintf("%x", bytes), nil
}

func (keyPair *ecdsaKeyPair) GetPrivateKey() *ecdsa.PrivateKey {
	return keyPair.privateKey
}

func (keyPair *ecdsaKeyPair) GetPublicKey() *ecdsa.PublicKey {
	return keyPair.publicKey
}

// privateKeyFromBytes reconstructs an ECDSA private key from a given byte slice.
// It returns the resulting private key and an error if the conversion fails.
func privateKeyFromBytes(bytes []byte) (*ecdsa.PrivateKey, error) {
	privateKey, err := x509.ParseECPrivateKey(bytes)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// publicKeyFromBytes reconstructs an ECDSA public key from a given byte slice.
// It returns the resulting public key and an error if the conversion fails.
func publicKeyFromBytes(bytes []byte) (*ecdsa.PublicKey, error) {
	publicKey, err := x509.ParsePKIXPublicKey(bytes)
	if err != nil {
		return nil, err
	}

	return publicKey.(*ecdsa.PublicKey), nil
}
