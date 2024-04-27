package transaction

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"

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

type TransactionWrapper struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func SerializeTransaction(tx Transaction) ([]byte, error) {
	data, err := tx.Serialize()
	if err != nil {
		return nil, err
	}

	wrapper := TransactionWrapper{
		Type: reflect.TypeOf(tx).Elem().Name(),
		Data: data,
	}

	return json.Marshal(wrapper)
}

func DeserializeTransaction(data []byte) (Transaction, error) {
	var wrapper TransactionWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	var tx Transaction
	switch wrapper.Type {
	case "AddFileTransaction":
		tx = &AddFileTransaction{}
	case "DeleteFileTransaction":
		tx = &DeleteFileTransaction{}
	default:
		return nil, fmt.Errorf("unknown transaction type: %s", wrapper.Type)
	}

	if err := json.Unmarshal(wrapper.Data, tx); err != nil {
		return nil, err
	}

	return tx, nil
}
