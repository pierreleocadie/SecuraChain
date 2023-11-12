package main

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Transaction is the struct that represents a transaction
// Random ID is generated for each transaction
// Timestamp is the time at which the transaction is created
type Transaction struct {
	ID        uuid.UUID `json:"id"`
	EmittedBy peer.ID   `json:"emittedBy"`
	Timestamp int64     `json:"timestamp"`
}

func generateTransaction(hostID peer.ID) []byte {
	trx := &Transaction{
		ID:        uuid.New(),
		EmittedBy: hostID,
		Timestamp: time.Now().Unix(),
	}
	b, _ := json.Marshal(trx)
	return b
}
