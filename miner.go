package main

import (
	"crypto/sha256"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	ecdsaSC "github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

type Header struct {
	MinedBy      peer.ID `json:"minedBy"`
	MinerAddress []byte  `json:"minerAddress"`
	Signature    []byte  `json:"signature"`
}

type Block struct {
	Header       Header        `json:"header"`
	ID           uuid.UUID     `json:"id"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
}

func addTrxToPool(trx []byte, trxPool *[]Transaction) {
	// Convert the transaction to a struct
	trxStruct := Transaction{}
	err := json.Unmarshal(trx, &trxStruct)
	if err != nil {
		log.Printf("Error unmarshalling transaction: %v", err)
		return
	}
	// Append the transaction to the pool
	*trxPool = append(*trxPool, trxStruct)
}

func resetPool(trxPool *[]Transaction) {
	*trxPool = []Transaction{}
}

func generateBlock(trxPool *[]Transaction, hostID peer.ID, keyPair ecdsaSC.KeyPair) []byte {
	// If there are 3 transactions in the pool, create a block
	defer resetPool(trxPool)
	if len(*trxPool) >= 3 {
		trxPoolBytes, err := json.Marshal(trxPool)
		if err != nil {
			log.Printf("Error marshalling transaction pool: %v", err)
			return nil
		}
		trxPoolHash := sha256.Sum256(trxPoolBytes)
		blockSignature, err := keyPair.Sign(trxPoolHash[:])
		if err != nil {
			log.Printf("Error signing block: %v", err)
			return nil
		}
		minerAddress, err := keyPair.PublicKeyToBytes()
		if err != nil {
			log.Printf("Error while converting public key to bytes: %v", err)
			return nil
		}
		header := Header{
			MinedBy:      hostID,
			MinerAddress: minerAddress,
			Signature:    blockSignature,
		}
		block := &Block{
			Header:       header,
			ID:           uuid.New(),
			Timestamp:    time.Now().Unix(),
			Transactions: *trxPool,
		}
		b, _ := json.Marshal(block)
		return b
	}
	return nil
}

func ValidateBlock(block []byte) bool {
	// Convert the block to a struct
	blockStruct := Block{}
	err := json.Unmarshal(block, &blockStruct)
	if err != nil {
		log.Printf("Error unmarshalling block: %v", err)
		return false
	}
	// Validate the block
	trxPoolBytes, err := json.Marshal(blockStruct.Transactions)
	if err != nil {
		log.Printf("Error marshalling transaction pool: %v", err)
		return false
	}
	trxPoolHash := sha256.Sum256(trxPoolBytes)
	minerPubKey, err := ecdsaSC.PublicKeyFromBytes(blockStruct.Header.MinerAddress)
	if err != nil {
		log.Printf("Error while converting public key from bytes: %v", err)
		return false
	}
	valid := ecdsaSC.VerifySignature(minerPubKey, trxPoolHash[:], blockStruct.Header.Signature)
	return valid
}
