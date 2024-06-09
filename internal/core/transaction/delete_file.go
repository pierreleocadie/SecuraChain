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
	OwnerAddress    []byte  `json:"ownerAddress"` // Owner address - ECDSA public key
	FileCid         cid.Cid `json:"fileCID"`      // File CID
	BaseTransaction         // embed BaseTransaction struct to inherit methods
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

	transaction.BaseTransaction.TransactionSignature, err = keyPair.Sign(transactionHash[:])
	if err != nil {
		return nil
	}

	return transaction
}

func DeserializeDeleteFileTransaction(data []byte) (*DeleteFileTransaction, error) {
	var transaction DeleteFileTransaction
	err := json.Unmarshal(data, &transaction)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}
