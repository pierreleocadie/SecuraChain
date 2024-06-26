package transaction

import (
	"crypto/sha256"
	"fmt"

	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

type TransactionVerifier struct{}

func (v TransactionVerifier) Verify(tx Transaction, signature []byte, publicKey []byte) error {
	data, err := tx.ToBytesWithoutSignature()
	if err != nil {
		return fmt.Errorf("failed to get transaction bytes without signature: %w", err)
	}
	fmt.Println("DEBUG HERE : ", data)
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
