package transaction

import (
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// ClientAnnouncementFactory implements the TransactionFactory interface
type ClientAnnouncementFactory struct{}

func (f *ClientAnnouncementFactory) CreateTransaction(data []byte) (Transaction, error) {
	return DeserializeClientAnnouncement(data)
}

type ClientAnnouncement struct {
	AnnouncementId        uuid.UUID `json:"announcementId"`        // Announcement ID - UUID
	OwnerAddress          []byte    `json:"ownerAddress"`          // Owner address - ECDSA public key
	Filename              []byte    `json:"filename"`              // Encrypted filename
	Extension             []byte    `json:"extension"`             // Encrypted extension
	FileSize              uint64    `json:"fileSize"`              // File size
	Checksum              []byte    `json:"checksum"`              // Checksum - SHA256
	OwnerSignature        []byte    `json:"ownerSignature"`        // Owner signature - ECDSA signature
	AnnouncementTimestamp int64     `json:"announcementTimestamp"` // Announcement timestamp - Unix timestamp
	TransactionVerifier             // embed TransactionVerifier struct to inherit VerifyTransaction method
}

func (a *ClientAnnouncement) Serialize() ([]byte, error) {
	return json.Marshal(a)
}

// Override SpecificData for ClientAnnouncement
func (a *ClientAnnouncement) SpecificData() ([]byte, error) {
	// Remove signature and serialize
	signature := a.OwnerSignature
	a.OwnerSignature = []byte{}
	defer func() { a.OwnerSignature = signature }() // Restore after serialization

	return json.Marshal(a)
}

func NewClientAnnouncement(keyPair ecdsa.KeyPair, filename []byte, extension []byte, fileSize uint64, checksum []byte) *ClientAnnouncement {
	ownerAddressBytes, err := keyPair.PublicKeyToBytes()
	if err != nil {
		return nil
	}

	announcement := &ClientAnnouncement{
		AnnouncementId:        uuid.New(),
		OwnerAddress:          ownerAddressBytes,
		Filename:              filename,
		Extension:             extension,
		FileSize:              fileSize,
		Checksum:              checksum,
		AnnouncementTimestamp: time.Now().Unix(),
	}

	announcementBytes, err := json.Marshal(announcement)
	if err != nil {
		return nil
	}
	announcementHash := sha256.Sum256(announcementBytes)

	announcement.OwnerSignature, err = keyPair.Sign(announcementHash[:])
	if err != nil {
		return nil
	}
	return announcement
}

func DeserializeClientAnnouncement(data []byte) (*ClientAnnouncement, error) {
	var announcement ClientAnnouncement
	err := json.Unmarshal(data, &announcement)
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}
