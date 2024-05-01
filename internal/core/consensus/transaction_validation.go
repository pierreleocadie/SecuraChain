package consensus

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// TransactionValidator interface
type TransactionValidator interface {
	Validate(tx transaction.Transaction) bool
}

func ValidateTransaction(tx transaction.Transaction) bool {
	var validator TransactionValidator

	switch tx.(type) {
	case *transaction.AddFileTransaction:
		validator = &AddFileTransactionValidator{}
	case *transaction.DeleteFileTransaction:
		validator = &DeleteFileTransactionValidator{}
	default:
		// Unknown transaction type
		fmt.Printf("Transaction validation failed: Unknown transaction type")
		return false
	}

	return validator.Validate(tx)
}

// AddFileTransactionValidator for AddFileTransaction
type AddFileTransactionValidator struct{}

func (v *AddFileTransactionValidator) Validate(tx transaction.Transaction) bool { //nolint: funlen
	addFileTransaction, ok := tx.(*transaction.AddFileTransaction)
	if !ok {
		fmt.Printf("Transaction validation failed: Transaction is not an AddFileTransaction")
		return false
	}

	// Verify that AnnouncementID and TransactionID are valid UUIDs
	if _, err := uuid.Parse(addFileTransaction.AnnouncementID.String()); err != nil {
		fmt.Printf("Transaction validation failed: AnnouncementID is not a valid UUID")
		return false
	}
	if _, err := uuid.Parse(addFileTransaction.TransactionID.String()); err != nil {
		fmt.Printf("Transaction validation failed: TransactionID is not a valid UUID")
		return false
	}

	if addFileTransaction.AnnouncementID == addFileTransaction.TransactionID {
		fmt.Printf("Transaction validation failed: AnnouncementID and TransactionID must be different")
		return false
	}

	// Verify that OwnerAddress and NodeAddress are valid ECDSA public keys
	if _, err := ecdsa.PublicKeyFromBytes(addFileTransaction.OwnerAddress); err != nil {
		fmt.Printf("Transaction validation failed: OwnerAddress is not a valid ECDSA public key")
		return false
	}
	if _, err := ecdsa.PublicKeyFromBytes(addFileTransaction.NodeAddress); err != nil {
		fmt.Printf("Transaction validation failed: NodeAddress is not a valid ECDSA public key")
		return false
	}

	if bytes.Equal(addFileTransaction.OwnerAddress, addFileTransaction.NodeAddress) {
		fmt.Printf("Transaction validation failed: OwnerAddress and NodeAddress must be different")
		return false
	}

	// Verify that checksum is a valid SHA256 hash
	if len(addFileTransaction.Checksum) != sha256.Size {
		fmt.Printf("Transaction validation failed: Checksum is not a valid SHA256 hash")
		return false
	}

	// Verify that NodeID is valid peer.ID
	if !isValidPeerID(addFileTransaction.NodeID) {
		fmt.Printf("Transaction validation failed: NodeCID is not a valid CID")
		return false
	}

	// Verify that IPFSClientNodeAddrInfo and IPFSStorageNodeAddrInfo are valid peer.AddrInfo
	if addFileTransaction.IPFSClientNodeAddrInfo.ID == "" || addFileTransaction.IPFSStorageNodeAddrInfo.ID == "" {
		fmt.Printf("Transaction validation failed: IPFSClientNodeAddrInfo and IPFSStorageNodeAddrInfo must have valid peer.IDs")
		return false
	}

	if addFileTransaction.IPFSClientNodeAddrInfo.ID == addFileTransaction.IPFSStorageNodeAddrInfo.ID {
		fmt.Printf("Transaction validation failed: IPFSClientNodeAddrInfo and IPFSStorageNodeAddrInfo must have different peer.IDs")
		return false
	}

	if addFileTransaction.IPFSStorageNodeAddrInfo.ID == addFileTransaction.NodeID {
		fmt.Printf("Transaction validation failed: IPFSStorageNodeAddrInfo and NodeID must have different peer.IDs")
		return false
	}

	if !isValidPeerID(addFileTransaction.IPFSClientNodeAddrInfo.ID) || !isValidPeerID(addFileTransaction.IPFSStorageNodeAddrInfo.ID) {
		fmt.Printf("Transaction validation failed: IPFSClientNodeAddrInfo and IPFSStorageNodeAddrInfo must have valid peer.IDs")
		return false
	}

	// Verify that FileCID is valid CID
	if !isValidCID(addFileTransaction.FileCid) {
		fmt.Printf("Transaction validation failed: FileCid is not a valid CID")
		return false
	}

	// Verify timestamps order (AnnouncementTimestamp < TransactionTimestamp)
	if !isValidTimestampOrder(addFileTransaction.AnnouncementTimestamp, addFileTransaction.TransactionTimestamp) {
		fmt.Printf("Transaction validation failed: Timestamps are not in the correct order")
		return false
	}

	// Verify that filename and extension have been provided
	if len(addFileTransaction.Filename) == 0 || len(addFileTransaction.Extension) == 0 {
		fmt.Printf("Transaction validation failed: Filename and extension must be provided")
		return false
	}

	// Verify the client announcement step
	clientAnnouncement := &transaction.ClientAnnouncement{
		AnnouncementID:         addFileTransaction.AnnouncementID,
		OwnerAddress:           addFileTransaction.OwnerAddress,
		IPFSClientNodeAddrInfo: addFileTransaction.IPFSClientNodeAddrInfo,
		FileCid:                addFileTransaction.FileCid,
		Filename:               addFileTransaction.Filename,
		Extension:              addFileTransaction.Extension,
		FileSize:               addFileTransaction.FileSize,
		Checksum:               addFileTransaction.Checksum,
		OwnerSignature:         addFileTransaction.OwnerSignature,
		AnnouncementTimestamp:  addFileTransaction.AnnouncementTimestamp,
	}

	if !clientAnnouncement.VerifyTransaction(clientAnnouncement, clientAnnouncement.OwnerSignature, clientAnnouncement.OwnerAddress) {
		fmt.Printf("Transaction validation failed: Client announcement signature is invalid")
		return false
	}

	// Verify the transaction step
	if !addFileTransaction.VerifyTransaction(addFileTransaction, addFileTransaction.TransactionSignature, addFileTransaction.NodeAddress) {
		fmt.Printf("Transaction validation failed: Transaction signature is invalid")
		return false
	}

	return true
}

// DeleteFileTransactionValidator for DeleteFileTransaction
type DeleteFileTransactionValidator struct{}

func (v *DeleteFileTransactionValidator) Validate(tx transaction.Transaction) bool {
	deleteFileTransaction, ok := tx.(*transaction.DeleteFileTransaction)
	if !ok {
		return false
	}

	// Verify that TransactionID is a valid UUID
	if _, err := uuid.Parse(deleteFileTransaction.TransactionID.String()); err != nil {
		return false
	}

	// Verify that OwnerAddress is a valid ECDSA public key
	if _, err := ecdsa.PublicKeyFromBytes(deleteFileTransaction.OwnerAddress); err != nil {
		return false
	}

	// Verify that FileCID is a valid CID
	if !isValidCID(deleteFileTransaction.FileCid) {
		return false
	}

	// Verify TransactionTimestamp
	if deleteFileTransaction.TransactionTimestamp <= 0 {
		return false
	}

	if deleteFileTransaction.TransactionTimestamp >= time.Now().UTC().Unix() {
		return false
	}

	// Verify the transaction signature
	return true
}

// isValidTimestampOrder checks if the provided timestamps are in the correct chronological order
func isValidTimestampOrder(timestamps ...int64) bool {
	for i := 0; i < len(timestamps)-1; i++ {
		if timestamps[i] >= timestamps[i+1] {
			return false
		}
	}
	return true
}

func isValidCID(c cid.Cid) bool {
	// Check if the CID is defined. An undefined CID is invalid.
	if !c.Defined() {
		return false
	}

	if c.Version() != 0 && c.Version() != 1 {
		return false
	}

	return true
}

func isValidPeerID(p peer.ID) bool {
	// Check if the peer.ID is valid
	if _, err := peer.Decode(p.String()); err != nil {
		return false
	}

	return true
}
