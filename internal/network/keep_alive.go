package network

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Packet struct {
	ID        uuid.UUID `json:"id"`
	EmittedBy peer.ID   `json:"emittedBy"`
	Timestamp int64     `json:"timestamp"`
}

func GeneratePacket(hostID peer.ID) []byte {
	trx := &Packet{
		ID:        uuid.New(),
		EmittedBy: hostID,
		Timestamp: time.Now().UTC().Unix(),
	}
	b, _ := json.Marshal(trx)
	return b
}
