package transaction

import (
	"crypto/sha256"
	"encoding/json"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// FileTransferHttpRequestFactory implements the TransactionFactory interface
type FileTransferHttpRequestFactory struct{}

func (f *FileTransferHttpRequestFactory) CreateTransaction(data []byte) (Transaction, error) {
	return DeserializeFileTransferHttpRequest(data)
}

type FileTransferHttpRequest struct {
	FileTransferId        uuid.UUID             `json:"fileTransferId" binding:"required"`        // File transfer ID - UUID
	AnnouncementId        uuid.UUID             `json:"announcementId" binding:"required"`        // Announcement ID - UUID
	OwnerAddress          []byte                `json:"ownerAddress" binding:"required"`          // Owner address - ECDSA public key
	Filename              []byte                `json:"filename" binding:"required"`              // Encrypted filename
	Extension             []byte                `json:"extension" binding:"required"`             // Encrypted extension
	FileSize              uint64                `json:"fileSize" binding:"required"`              // File size
	Checksum              []byte                `json:"checksum" binding:"required"`              // Checksum - SHA256
	OwnerSignature        []byte                `json:"ownerSignature" binding:"required"`        // Owner signature - ECDSA signature
	AnnouncementTimestamp int64                 `json:"announcementTimestamp" binding:"required"` // Announcement timestamp - Unix timestamp
	ResponseId            uuid.UUID             `json:"responseId" binding:"required"`            // Response ID - UUID
	NodeAddress           []byte                `json:"nodeAddress" binding:"required"`           // Node address - ECDSA public key
	NodeCID               []byte                `json:"nodeCID" binding:"required"`               // Node CID - SHA256
	NodeSignature         []byte                `json:"nodeSignature" binding:"required"`         // Node signature - ECDSA signature
	ResponseTimestamp     int64                 `json:"responseTimestamp" binding:"required"`     // Response timestamp - Unix timestamp
	File                  *multipart.FileHeader `form:"file" json:"file" binding:"required"`      // File - multipart.FileHeader
	FileTransferSignature []byte                `json:"fileTransferSignature" binding:"required"` // File transfer signature - ECDSA signature
	FileTransferTimestamp int64                 `json:"fileTransferTimestamp" binding:"required"` // File transfer timestamp - Unix timestamp
	TransactionVerifier                         // embed TransactionVerifier struct to inherit VerifyTransaction method
}

func (f *FileTransferHttpRequest) Serialize() ([]byte, error) {
	return json.Marshal(f)
}

// Override SpecificData for FileTransferHttpRequest
func (f *FileTransferHttpRequest) SpecificData() ([]byte, error) {
	// Remove signature from file transfer before verifying
	signature := f.FileTransferSignature
	f.FileTransferSignature = []byte{}

	defer func() { f.FileTransferSignature = signature }() // Restore after serialization

	return json.Marshal(f)
}

func NewFileTransferHttpRequest(announcement *ClientAnnouncement, response *StorageNodeResponse, file *multipart.FileHeader, keyPair ecdsa.KeyPair) *FileTransferHttpRequest {
	fileTransfer := &FileTransferHttpRequest{
		FileTransferId:        uuid.New(),
		AnnouncementId:        announcement.AnnouncementId,
		OwnerAddress:          announcement.OwnerAddress,
		Filename:              announcement.Filename,
		Extension:             announcement.Extension,
		FileSize:              announcement.FileSize,
		Checksum:              announcement.Checksum,
		OwnerSignature:        announcement.OwnerSignature,
		AnnouncementTimestamp: announcement.AnnouncementTimestamp,
		ResponseId:            response.ResponseId,
		NodeAddress:           response.NodeAddress,
		NodeCID:               response.NodeCID,
		NodeSignature:         response.NodeSignature,
		ResponseTimestamp:     response.ResponseTimestamp,
		File:                  file,
		FileTransferTimestamp: time.Now().Unix(),
	}

	fileTransferBytes, err := json.Marshal(fileTransfer)
	if err != nil {
		return nil
	}

	fileTransferHash := sha256.Sum256(fileTransferBytes)

	fileTransfer.FileTransferSignature, err = keyPair.Sign(fileTransferHash[:])
	if err != nil {
		return nil
	}

	return fileTransfer
}

func DeserializeFileTransferHttpRequest(data []byte) (*FileTransferHttpRequest, error) {
	var fileTransfer FileTransferHttpRequest
	err := json.Unmarshal(data, &fileTransfer)
	if err != nil {
		return nil, err
	}
	return &fileTransfer, nil
}
