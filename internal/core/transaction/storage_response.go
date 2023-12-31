package transaction

import (
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// StorageNodeResponseFactory implements the TransactionFactory interface
type StorageNodeResponseFactory struct{}

func (f *StorageNodeResponseFactory) CreateTransaction(data []byte) (Transaction, error) {
	return DeserializeStorageNodeResponse(data)
}

type StorageNodeResponse struct {
	ResponseID            uuid.UUID `json:"responseID"`            // Response ID - UUID
	NodeAddress           []byte    `json:"nodeAddress"`           // Node address - ECDSA public key
	NodeID                peer.ID   `json:"nodeID"`                // Node ID
	APIEndpoint           string    `json:"apiEndpoint"`           // API endpoint - URL for file transfer binding:"required"
	NodeSignature         []byte    `json:"nodeSignature"`         // Node signature - ECDSA signature
	ResponseTimestamp     int64     `json:"responseTimestamp"`     // Response timestamp - Unix timestamp
	AnnouncementID        uuid.UUID `json:"announcementID"`        // Announcement ID - UUID
	OwnerAddress          []byte    `json:"ownerAddress"`          // Owner address - ECDSA public key
	Filename              []byte    `json:"filename"`              // Encrypted filename
	Extension             []byte    `json:"extension"`             // Encrypted extension
	FileSize              uint64    `json:"fileSize"`              // File size
	Checksum              []byte    `json:"checksum"`              // Checksum - SHA256
	OwnerSignature        []byte    `json:"ownerSignature"`        // Owner signature - ECDSA signature
	AnnouncementTimestamp int64     `json:"announcementTimestamp"` // Announcement timestamp - Unix timestamp
	Verifier                        // embed TransactionVerifier struct to inherit VerifyTransaction method
}

func (r *StorageNodeResponse) Serialize() ([]byte, error) {
	return json.Marshal(r)
}

// Override SpecificData for StorageNodeResponse
func (r *StorageNodeResponse) SpecificData() ([]byte, error) {
	// Remove signature from response before verifying
	signature := r.NodeSignature
	r.NodeSignature = nil

	defer func() { r.NodeSignature = signature }() // Restore after serialization

	return json.Marshal(r)
}

func NewStorageNodeResponse(nodeAddress ecdsa.KeyPair, nodeID peer.ID, apiEndpoint string, announcement *ClientAnnouncement) *StorageNodeResponse {
	nodeAddressBytes, err := nodeAddress.PublicKeyToBytes()
	if err != nil {
		return nil
	}

	response := &StorageNodeResponse{
		ResponseID:            uuid.New(),
		NodeAddress:           nodeAddressBytes,
		NodeID:                nodeID,
		APIEndpoint:           apiEndpoint,
		ResponseTimestamp:     time.Now().Unix(),
		AnnouncementID:        announcement.AnnouncementID,
		OwnerAddress:          announcement.OwnerAddress,
		Filename:              announcement.Filename,
		Extension:             announcement.Extension,
		FileSize:              announcement.FileSize,
		Checksum:              announcement.Checksum,
		OwnerSignature:        announcement.OwnerSignature,
		AnnouncementTimestamp: announcement.AnnouncementTimestamp,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return nil
	}

	responseHash := sha256.Sum256(responseBytes)

	response.NodeSignature, err = nodeAddress.Sign(responseHash[:])
	if err != nil {
		return nil
	}
	return response
}

func DeserializeStorageNodeResponse(data []byte) (*StorageNodeResponse, error) {
	var response StorageNodeResponse
	err := json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
