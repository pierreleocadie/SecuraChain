package transaction

import (
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

type DeleteFileTransaction struct {
	BaseTransaction
	OwnerAddress []byte  `json:"ownerAddress"` // Owner address - ECDSA public key
	FileCid      cid.Cid `json:"fileCID"`      // File CID
}

func NewDeleteFileTransaction(keyPair ecdsa.KeyPair, fileCid cid.Cid) *DeleteFileTransaction {
	ownerAddressBytes, err := keyPair.PublicKeyToBytes()
	if err != nil {
		return nil
	}

	transaction := &DeleteFileTransaction{
		OwnerAddress: ownerAddressBytes,
		FileCid:      fileCid,
		BaseTransaction: BaseTransaction{
			TransactionID:        uuid.New(),
			TransactionTimestamp: time.Now().UTC().Unix(),
		},
	}

	transactionBytes, err := json.Marshal(transaction)
	if err != nil {
		return nil
	}
	transactionHash := sha256.Sum256(transactionBytes)

	transaction.TransactionSignature, err = keyPair.Sign(transactionHash[:])
	if err != nil {
		return nil
	}

	return transaction
}

func (t DeleteFileTransaction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t DeleteFileTransaction) ToBytesWithoutSignature() ([]byte, error) {
	// Remove signature from transaction before verifying
	signature := t.TransactionSignature
	t.TransactionSignature = nil

	defer func() { t.TransactionSignature = signature }() // Restore after serialization

	return json.Marshal(t)
}

func DeserializeDeleteFileTransaction(data []byte) (*DeleteFileTransaction, error) {
	var transaction DeleteFileTransaction
	err := json.Unmarshal(data, &transaction)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}
