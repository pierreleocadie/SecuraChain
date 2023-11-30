package core

import (
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

type StorageNodeResponse struct {
	ResponseId            uuid.UUID `json:"responseId"`            // Response ID - UUID
	NodeAddress           []byte    `json:"nodeAddress"`           // Node address - ECDSA public key
	NodeCID               []byte    `json:"nodeCID"`               // Node CID - SHA256
	APIEndpoint           string    `json:"apiEndpoint"`           // API endpoint - URL for file transfer binding:"required"
	NodeSignature         []byte    `json:"nodeSignature"`         // Node signature - ECDSA signature
	ResponseTimestamp     int64     `json:"responseTimestamp"`     // Response timestamp - Unix timestamp
	AnnouncementId        uuid.UUID `json:"announcementId"`        // Announcement ID - UUID
	OwnerAddress          []byte    `json:"ownerAddress"`          // Owner address - ECDSA public key
	Filename              []byte    `json:"filename"`              // Encrypted filename
	Extension             []byte    `json:"extension"`             // Encrypted extension
	FileSize              uint64    `json:"fileSize"`              // File size
	Checksum              []byte    `json:"checksum"`              // Checksum - SHA256
	OwnerSignature        []byte    `json:"ownerSignature"`        // Owner signature - ECDSA signature
	AnnouncementTimestamp int64     `json:"announcementTimestamp"` // Announcement timestamp - Unix timestamp
}

func NewStorageNodeResponse(nodeAddress ecdsa.KeyPair, nodeCID []byte, apiEndpoint string, announcement *ClientAnnouncement) *StorageNodeResponse {
	nodeAddressBytes, err := nodeAddress.PublicKeyToBytes()
	if err != nil {
		return nil
	}

	response := &StorageNodeResponse{
		ResponseId:            uuid.New(),
		NodeAddress:           nodeAddressBytes,
		NodeCID:               nodeCID,
		APIEndpoint:           apiEndpoint,
		ResponseTimestamp:     time.Now().Unix(),
		AnnouncementId:        announcement.AnnouncementId,
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

func (r *StorageNodeResponse) Serialize() ([]byte, error) {
	return json.Marshal(r)
}

func DeserializeStorageNodeResponse(data []byte) (*StorageNodeResponse, error) {
	var response StorageNodeResponse
	err := json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func VerifyStorageNodeResponse(response *StorageNodeResponse) bool {
	// Remove signature from response before verifying
	nodeSignature := response.NodeSignature
	response.NodeSignature = []byte{}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return false
	}
	responseHash := sha256.Sum256(responseBytes)

	// Restore signature
	response.NodeSignature = nodeSignature

	nodeAddress, err := ecdsa.PublicKeyFromBytes(response.NodeAddress)
	if err != nil {
		return false
	}

	return ecdsa.VerifySignature(nodeAddress, responseHash[:], nodeSignature)
}
