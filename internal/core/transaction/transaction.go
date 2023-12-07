package transaction

import (
	"crypto/sha256"

	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// TransactionVerifier is the base struct for verifying transactions
type Verifier struct{}

// SpecificData method should be overridden by each specific transaction type
func (t *Verifier) SpecificData() ([]byte, error) {
	return nil, nil // Default implementation, should be overridden
}

func (t *Verifier) VerifyTransaction(tx Transaction, signature []byte, publicKey []byte) bool {
	data, err := tx.SpecificData()
	if err != nil {
		return false
	}

	hash := sha256.Sum256(data)

	ownerAddr, err := ecdsa.PublicKeyFromBytes(publicKey)
	if err != nil {
		return false
	}

	return ecdsa.VerifySignature(ownerAddr, hash[:], signature)
}

type Transaction interface {
	Serialize() ([]byte, error)
	SpecificData() ([]byte, error)
}

// TransactionFactory is the interface for creating transactions
type Factory interface {
	CreateTransaction(data []byte) (Transaction, error)
}

func DeserializeTransaction(data []byte, factory Factory) (Transaction, error) {
	return factory.CreateTransaction(data)
}
