package transaction

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Transaction interface {
	Serialize() ([]byte, error)
	ToBytesWithoutSignature() ([]byte, error)
	Verify(signature []byte, publicKey []byte) error
}

type TransactionWrapper struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func SerializeTransaction(tx Transaction) ([]byte, error) {
	data, err := json.Marshal(tx)
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
