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
	TransactionID        uuid.UUID `json:"transactionID"`        // Transaction ID - UUID
	OwnerAddress         []byte    `json:"ownerAddress"`         // Owner address - ECDSA public key
	FileCid              cid.Cid   `json:"fileCID"`              // File CID
	TransactionSignature []byte    `json:"transactionSignature"` // Transaction signature - ECDSA signature
	TransactionTimestamp int64     `json:"transactionTimestamp"` // Transaction timestamp - Unix timestamp
	Verifier                       // embed TransactionVerifier struct to inherit VerifyTransaction method
}

func (t *DeleteFileTransaction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

// Override SpecificData for DeleteFileTransaction
func (t *DeleteFileTransaction) SpecificData() ([]byte, error) {
	// Remove signature from transaction before verifying
	signature := t.TransactionSignature
	t.TransactionSignature = []byte{}

	defer func() { t.TransactionSignature = signature }() // Restore after serialization

	return json.Marshal(t)
}

func NewDeleteFileTransaction(keyPair ecdsa.KeyPair, fileCid cid.Cid) *DeleteFileTransaction {
	ownerAddressBytes, err := keyPair.PublicKeyToBytes()
	if err != nil {
		return nil
	}

	transaction := &DeleteFileTransaction{
		TransactionID:        uuid.New(),
		OwnerAddress:         ownerAddressBytes,
		FileCid:              fileCid,
		TransactionTimestamp: time.Now().Unix(),
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

func DeserializeDeleteFileTransaction(data []byte) (*DeleteFileTransaction, error) {
	var transaction DeleteFileTransaction
	err := json.Unmarshal(data, &transaction)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}
