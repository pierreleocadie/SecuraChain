package transaction

import (
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// AddFileTransactionFactory implements the TransactionFactory interface
type AddFileTransactionFactory struct{}

func (f *AddFileTransactionFactory) CreateTransaction(data []byte) (Transaction, error) {
	return DeserializeAddFileTransaction(data)
}

type AddFileTransaction struct {
	AnnouncementID        uuid.UUID `json:"announcementID"`        // Announcement ID - UUID
	OwnerAddress          []byte    `json:"ownerAddress"`          // Owner address - ECDSA public key
	Filename              []byte    `json:"filename"`              // Encrypted filename
	Extension             []byte    `json:"extension"`             // Encrypted extension
	FileSize              uint64    `json:"fileSize"`              // File size
	Checksum              []byte    `json:"checksum"`              // Checksum - SHA256
	OwnerSignature        []byte    `json:"ownerSignature"`        // Owner signature - ECDSA signature
	AnnouncementTimestamp int64     `json:"announcementTimestamp"` // Announcement timestamp - Unix timestamp
	ResponseID            uuid.UUID `json:"responseID"`            // Response ID - UUID
	NodeAddress           []byte    `json:"nodeAddress"`           // Node address - ECDSA public key
	NodeCID               cid.Cid   `json:"nodeCID"`               // Node CID
	NodeSignature         []byte    `json:"nodeSignature"`         // Node signature - ECDSA signature
	ResponseTimestamp     int64     `json:"responseTimestamp"`     // Response timestamp - Unix timestamp
	FileTransferID        uuid.UUID `json:"fileTransferID"`        // File transfer ID - UUID
	FileTransferSignature []byte    `json:"fileTransferSignature"` // File transfer signature - ECDSA signature
	FileTransferTimestamp int64     `json:"fileTransferTimestamp"` // File transfer timestamp - Unix timestamp
	FileCID               cid.Cid   `json:"fileCID"`               // File CID
	TransactionID         uuid.UUID `json:"transactionID"`         // Transaction ID - UUID
	UserReliabilityIssue  bool      `json:"userReliabilityIssue"`  // User reliability issue - boolean
	TransactionSignature  []byte    `json:"transactionSignature"`  // Transaction signature - ECDSA signature
	TransactionTimestamp  int64     `json:"transactionTimestamp"`  // Transaction timestamp - Unix timestamp
	Verifier                        // embed TransactionVerifier struct to inherit VerifyTransaction method
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

func NewAddFileTransaction(announcement *ClientAnnouncement, response *StorageNodeResponse,
	fileTransfer *FileTransferHTTPRequest, fileCID cid.Cid,
	userReliabilityIssue bool, keyPair ecdsa.KeyPair) *AddFileTransaction {
	transaction := &AddFileTransaction{
		TransactionID:         uuid.New(),
		AnnouncementID:        announcement.AnnouncementID,
		OwnerAddress:          announcement.OwnerAddress,
		Filename:              announcement.Filename,
		Extension:             announcement.Extension,
		FileSize:              announcement.FileSize,
		Checksum:              announcement.Checksum,
		OwnerSignature:        announcement.OwnerSignature,
		AnnouncementTimestamp: announcement.AnnouncementTimestamp,
		ResponseID:            response.ResponseID,
		NodeAddress:           response.NodeAddress,
		NodeCID:               response.NodeCID,
		NodeSignature:         response.NodeSignature,
		ResponseTimestamp:     response.ResponseTimestamp,
		FileTransferID:        fileTransfer.FileTransferID,
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
