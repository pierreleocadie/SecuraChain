package transaction

import (
	"crypto/sha256"
	"fmt"

	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

type TransactionVerifier struct{}

// Dumb implementation of Serialize
func (v TransactionVerifier) Serialize() ([]byte, error) {
	return nil, nil
}

// Dumb implementation of ToBytesWithoutSignature
func (v TransactionVerifier) ToBytesWithoutSignature() ([]byte, error) {
	return nil, nil
}

func (v TransactionVerifier) Verify(signature []byte, publicKey []byte) error {
	data, err := v.ToBytesWithoutSignature()
	if err != nil {
		return fmt.Errorf("failed to get transaction bytes without signature: %w", err)
	}

	hash := sha256.Sum256(data)

	addr, err := ecdsa.PublicKeyFromBytes(publicKey)
	if err != nil {
		return fmt.Errorf("failed to get public key from bytes: %w", err)
	}

	if !ecdsa.VerifySignature(addr, hash[:], signature) {
		return fmt.Errorf("transaction signature is invalid")
	}

	return nil
}
