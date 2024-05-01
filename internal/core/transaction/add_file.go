package transaction

import (
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

type AddFileTransaction struct {
	AnnouncementID          uuid.UUID     `json:"announcementID"`          // Announcement ID - UUID
	OwnerAddress            []byte        `json:"ownerAddress"`            // Owner address - ECDSA public key
	IPFSClientNodeAddrInfo  peer.AddrInfo `json:"ipfsClientNodeAddrInfo"`  // IPFS client node address info
	FileCid                 cid.Cid       `json:"fileCid"`                 // File CID
	Filename                []byte        `json:"filename"`                // Encrypted filename
	Extension               []byte        `json:"extension"`               // Encrypted extension
	FileSize                uint64        `json:"fileSize"`                // File size
	Checksum                []byte        `json:"checksum"`                // Checksum - SHA256
	OwnerSignature          []byte        `json:"ownerSignature"`          // Owner signature - ECDSA signature
	AnnouncementTimestamp   int64         `json:"announcementTimestamp"`   // Announcement timestamp - Unix timestamp
	NodeAddress             []byte        `json:"nodeAddress"`             // Node address - ECDSA public key
	NodeID                  peer.ID       `json:"nodeID"`                  // Node ID
	IPFSStorageNodeAddrInfo peer.AddrInfo `json:"ipfsStorageNodeAddrInfo"` // IPFS storage node address info
	TransactionID           uuid.UUID     `json:"transactionID"`           // Transaction ID - UUID
	UserReliabilityIssue    bool          `json:"userReliabilityIssue"`    // User reliability issue - boolean
	TransactionSignature    []byte        `json:"transactionSignature"`    // Transaction signature - ECDSA signature
	TransactionTimestamp    int64         `json:"transactionTimestamp"`    // Transaction timestamp - Unix timestamp
	Verifier                              // embed TransactionVerifier struct to inherit VerifyTransaction method
}

func (t *AddFileTransaction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

// Override SpecificData for AddFileTransaction
func (t *AddFileTransaction) SpecificData() ([]byte, error) {
	// Remove signature from transaction before verifying
	signature := t.TransactionSignature
	t.TransactionSignature = nil

	defer func() { t.TransactionSignature = signature }() // Restore after serialization

	return json.Marshal(t)
}

func NewAddFileTransaction(announcement *ClientAnnouncement, fileCid cid.Cid,
	userReliabilityIssue bool, keyPair ecdsa.KeyPair, nodeID peer.ID, storageNodeAddrInfo peer.AddrInfo) *AddFileTransaction {
	nodeAddressBytes, err := keyPair.PublicKeyToBytes()
	if err != nil {
		return nil
	}

	transaction := &AddFileTransaction{
		TransactionID:           uuid.New(),
		AnnouncementID:          announcement.AnnouncementID,
		OwnerAddress:            announcement.OwnerAddress,
		IPFSClientNodeAddrInfo:  announcement.IPFSClientNodeAddrInfo,
		FileCid:                 fileCid,
		Filename:                announcement.Filename,
		Extension:               announcement.Extension,
		FileSize:                announcement.FileSize,
		Checksum:                announcement.Checksum,
		OwnerSignature:          announcement.OwnerSignature,
		AnnouncementTimestamp:   announcement.AnnouncementTimestamp,
		NodeAddress:             nodeAddressBytes,
		IPFSStorageNodeAddrInfo: storageNodeAddrInfo,
		NodeID:                  nodeID,
		UserReliabilityIssue:    userReliabilityIssue,
		TransactionTimestamp:    time.Now().UTC().Unix(),
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

func DeserializeAddFileTransaction(data []byte) (*AddFileTransaction, error) {
	var transaction AddFileTransaction
	err := json.Unmarshal(data, &transaction)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}
