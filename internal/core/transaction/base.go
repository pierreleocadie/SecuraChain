package transaction

import (
	"github.com/google/uuid"
)

type BaseTransaction struct {
	TransactionVerifier            // embed TransactionVerifier struct to inherit methods
	TransactionID        uuid.UUID `json:"transactionID"`        // Transaction ID - UUID
	TransactionSignature []byte    `json:"transactionSignature"` // Transaction signature - ECDSA signature
	TransactionTimestamp int64     `json:"transactionTimestamp"` // Transaction timestamp - Unix timestamp
}
