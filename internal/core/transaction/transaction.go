package transaction

import (
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

type Transaction interface {
	Serialize() ([]byte, error)
	VerifyTransaction() bool
}

type AddFileTransaction struct {
	TransactionId         uuid.UUID `json:"transactionId"`         // Transaction ID - UUID
	AnnouncementId        uuid.UUID `json:"announcementId"`        // Announcement ID - UUID
	OwnerAddress          []byte    `json:"ownerAddress"`          // Owner address - ECDSA public key
	Filename              []byte    `json:"filename"`              // Encrypted filename
	Extension             []byte    `json:"extension"`             // Encrypted extension
	Checksum              []byte    `json:"checksum"`              // Checksum - SHA256
	OwnerSignature        []byte    `json:"ownerSignature"`        // Owner signature - ECDSA signature
	AnnouncementTimestamp int64     `json:"announcementTimestamp"` // Announcement timestamp - Unix timestamp
	ResponseId            uuid.UUID `json:"responseId"`            // Response ID - UUID
	NodeAddress           []byte    `json:"nodeAddress"`           // Node address - ECDSA public key
	NodeCID               []byte    `json:"nodeCID"`               // Node CID - SHA256
	NodeSignature         []byte    `json:"nodeSignature"`         // Node signature - ECDSA signature
	ResponseTimestamp     int64     `json:"responseTimestamp"`     // Response timestamp - Unix timestamp
	FileTransferId        uuid.UUID `json:"fileTransferId"`        // File transfer ID - UUID
	FileTransferSignature []byte    `json:"fileTransferSignature"` // File transfer signature - ECDSA signature
	FileTransferTimestamp int64     `json:"fileTransferTimestamp"` // File transfer timestamp - Unix timestamp
	FileCID               []byte    `json:"fileCID"`               // File CID - SHA256
	UserReliabilityIssue  bool      `json:"userReliabilityIssue"`  // User reliability issue - boolean
	TransactionSignature  []byte    `json:"transactionSignature"`  // Transaction signature - ECDSA signature
	TransactionTimestamp  int64     `json:"transactionTimestamp"`  // Transaction timestamp - Unix timestamp
}

func NewAddFileTransaction(announcement *ClientAnnouncement, response *StorageNodeResponse, fileTransfer *FileTransferHttpRequest, fileCID []byte, userReliabilityIssue bool, keyPair ecdsa.KeyPair) *AddFileTransaction {
	transaction := &AddFileTransaction{
		TransactionId:         uuid.New(),
		AnnouncementId:        announcement.AnnouncementId,
		OwnerAddress:          announcement.OwnerAddress,
		Filename:              announcement.Filename,
		Extension:             announcement.Extension,
		Checksum:              announcement.Checksum,
		OwnerSignature:        announcement.OwnerSignature,
		AnnouncementTimestamp: announcement.AnnouncementTimestamp,
		ResponseId:            response.ResponseId,
		NodeAddress:           response.NodeAddress,
		NodeCID:               response.NodeCID,
		NodeSignature:         response.NodeSignature,
		ResponseTimestamp:     response.ResponseTimestamp,
		FileTransferId:        fileTransfer.FileTransferId,
		FileTransferSignature: fileTransfer.FileTransferSignature,
		FileTransferTimestamp: fileTransfer.FileTransferTimestamp,
		FileCID:               fileCID,
		UserReliabilityIssue:  userReliabilityIssue,
		TransactionTimestamp:  time.Now().Unix(),
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

func (t *AddFileTransaction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t *AddFileTransaction) VerifyTransaction() bool {
	// Remove signature from transaction before verifying
	transactionSignature := t.TransactionSignature
	t.TransactionSignature = []byte{}

	transactionBytes, err := json.Marshal(t)
	if err != nil {
		return false
	}
	transactionHash := sha256.Sum256(transactionBytes)

	// Restore signature
	t.TransactionSignature = transactionSignature

	ownerAddress, err := ecdsa.PublicKeyFromBytes(t.OwnerAddress)
	if err != nil {
		return false
	}

	return ecdsa.VerifySignature(ownerAddress, transactionHash[:], transactionSignature)
}

type DeleteFileTransaction struct {
	TransactionId        uuid.UUID `json:"transactionId"`        // Transaction ID - UUID
	OwnerAddress         []byte    `json:"ownerAddress"`         // Owner address - ECDSA public key
	FileCID              []byte    `json:"fileCID"`              // File CID - SHA256
	TransactionSignature []byte    `json:"transactionSignature"` // Transaction signature - ECDSA signature
	TransactionTimestamp int64     `json:"transactionTimestamp"` // Transaction timestamp - Unix timestamp
}

func NewDeleteFileTransaction(keyPair ecdsa.KeyPair, fileCID []byte) *DeleteFileTransaction {
	ownerAddressBytes, err := keyPair.PublicKeyToBytes()
	if err != nil {
		return nil
	}

	transaction := &DeleteFileTransaction{
		TransactionId:        uuid.New(),
		OwnerAddress:         ownerAddressBytes,
		FileCID:              fileCID,
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

func (t *DeleteFileTransaction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t *DeleteFileTransaction) VerifyTransaction() bool {
	// Remove signature from transaction before verifying
	transactionSignature := t.TransactionSignature
	t.TransactionSignature = []byte{}

	transactionBytes, err := json.Marshal(t)
	if err != nil {
		return false
	}
	transactionHash := sha256.Sum256(transactionBytes)

	// Restore signature
	t.TransactionSignature = transactionSignature

	ownerAddress, err := ecdsa.PublicKeyFromBytes(t.OwnerAddress)
	if err != nil {
		return false
	}

	return ecdsa.VerifySignature(ownerAddress, transactionHash[:], transactionSignature)
}
