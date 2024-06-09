package transaction

import (
	"encoding/json"

	"github.com/google/uuid"
)

type BaseTransaction struct {
	TransactionVerifier            // embed TransactionVerifier struct to inherit methods
	TransactionID        uuid.UUID `json:"transactionID"`        // Transaction ID - UUID
	TransactionSignature []byte    `json:"transactionSignature"` // Transaction signature - ECDSA signature
	TransactionTimestamp int64     `json:"transactionTimestamp"` // Transaction timestamp - Unix timestamp
}

// Override Serialize from TransactionVerifier
func (t BaseTransaction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

// Override ToBytesWithoutSignature from TransactionVerifier
func (t BaseTransaction) ToBytesWithoutSignature() ([]byte, error) {
	// Remove signature from transaction before verifying
	signature := t.TransactionSignature
	t.TransactionSignature = nil

	defer func() { t.TransactionSignature = signature }() // Restore after serialization

	return json.Marshal(t)
}
