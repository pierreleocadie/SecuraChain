package transaction

import (
	"crypto/sha256"
	"encoding/json"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// FileTransferHttpRequestFactory implements the TransactionFactory interface
type FileTransferHTTPRequestFactory struct{}

func (f *FileTransferHTTPRequestFactory) CreateTransaction(data []byte) (Transaction, error) {
	return DeserializeFileTransferHTTPRequest(data)
}

type FileTransferHTTPRequest struct {
	FileTransferID        uuid.UUID             `json:"fileTransferID" binding:"required"`        // File transfer ID - UUID
	AnnouncementID        uuid.UUID             `json:"announcementID" binding:"required"`        // Announcement ID - UUID
	OwnerAddress          []byte                `json:"ownerAddress" binding:"required"`          // Owner address - ECDSA public key
	Filename              []byte                `json:"filename" binding:"required"`              // Encrypted filename
	Extension             []byte                `json:"extension" binding:"required"`             // Encrypted extension
	FileSize              uint64                `json:"fileSize" binding:"required"`              // File size
	Checksum              []byte                `json:"checksum" binding:"required"`              // Checksum - SHA256
	OwnerSignature        []byte                `json:"ownerSignature" binding:"required"`        // Owner signature - ECDSA signature
	AnnouncementTimestamp int64                 `json:"announcementTimestamp" binding:"required"` // Announcement timestamp - Unix timestamp
	ResponseID            uuid.UUID             `json:"responseID" binding:"required"`            // Response ID - UUID
	NodeAddress           []byte                `json:"nodeAddress" binding:"required"`           // Node address - ECDSA public key
	NodeID                peer.ID               `json:"nodeID" binding:"required"`                // Node CID - SHA256
	NodeSignature         []byte                `json:"nodeSignature" binding:"required"`         // Node signature - ECDSA signature
	ResponseTimestamp     int64                 `json:"responseTimestamp" binding:"required"`     // Response timestamp - Unix timestamp
	File                  *multipart.FileHeader `form:"file" json:"file" binding:"required"`      // File - multipart.FileHeader
	FileTransferSignature []byte                `json:"fileTransferSignature" binding:"required"` // File transfer signature - ECDSA signature
	FileTransferTimestamp int64                 `json:"fileTransferTimestamp" binding:"required"` // File transfer timestamp - Unix timestamp
	Verifier                                    // embed TransactionVerifier struct to inherit VerifyTransaction method
}

func (f *FileTransferHTTPRequest) Serialize() ([]byte, error) {
	return json.Marshal(f)
}

// Override SpecificData for FileTransferHttpRequest
func (f *FileTransferHTTPRequest) SpecificData() ([]byte, error) {
	// Remove signature from file transfer before verifying
	signature := f.FileTransferSignature
	f.FileTransferSignature = nil

	defer func() { f.FileTransferSignature = signature }() // Restore after serialization

	return json.Marshal(f)
}

func NewFileTransferHTTPRequest(announcement *ClientAnnouncement, response *StorageNodeResponse,
	file *multipart.FileHeader, keyPair ecdsa.KeyPair) *FileTransferHTTPRequest {
	fileTransfer := &FileTransferHTTPRequest{
		FileTransferID:        uuid.New(),
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
		NodeID:                response.NodeID,
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

func DeserializeFileTransferHTTPRequest(data []byte) (*FileTransferHTTPRequest, error) {
	var fileTransfer FileTransferHTTPRequest
	err := json.Unmarshal(data, &fileTransfer)
	if err != nil {
		return nil, err
	}
	return &fileTransfer, nil
}
