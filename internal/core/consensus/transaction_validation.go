package consensus

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

type TransactionValidatorFactory interface {
	GetValidator(tx transaction.Transaction) (TransactionValidator, error)
}

type DefaultTransactionValidatorFactory struct{}

func (f DefaultTransactionValidatorFactory) GetValidator(tx transaction.Transaction) (TransactionValidator, error) {
	switch tx.(type) {
	case *transaction.AddFileTransaction:
		return AddFileTransactionValidator{}, nil
	case *transaction.DeleteFileTransaction:
		return DeleteFileTransactionValidator{}, nil
	default:
		return nil, fmt.Errorf("transaction validation failed: Unknown transaction type")
	}
}

// TransactionValidator interface
type TransactionValidator interface {
	Validate(tx transaction.Transaction) error
}

func ValidateTransaction(tx transaction.Transaction) error {
	var validator TransactionValidator

	switch tx.(type) {
	case *transaction.AddFileTransaction:
		validator = AddFileTransactionValidator{}
	case *transaction.DeleteFileTransaction:
		validator = DeleteFileTransactionValidator{}
	default:
		// Unknown transaction type
		return fmt.Errorf("transaction validation failed: Unknown transaction type")
	}

	return validator.Validate(tx)
}

// AddFileTransactionValidator for AddFileTransaction
type AddFileTransactionValidator struct{}

func (v AddFileTransactionValidator) Validate(tx transaction.Transaction) error { //nolint: funlen
	addFileTransaction, ok := tx.(*transaction.AddFileTransaction)
	if !ok {
		return fmt.Errorf("transaction validation failed: Transaction is not an AddFileTransaction")
	}

	// Verify that AnnouncementID and TransactionID are valid UUIDs
	if _, err := uuid.Parse(addFileTransaction.AnnouncementID.String()); err != nil {
		return fmt.Errorf("transaction validation failed: AnnouncementID is not a valid UUID")
	}
	if _, err := uuid.Parse(addFileTransaction.TransactionID.String()); err != nil {
		return fmt.Errorf("transaction validation failed: TransactionID is not a valid UUID")
	}

	if addFileTransaction.AnnouncementID == addFileTransaction.TransactionID {
		return fmt.Errorf("transaction validation failed: AnnouncementID and TransactionID must be different")
	}

	// Verify that OwnerAddress and NodeAddress are valid ECDSA public keys
	if _, err := ecdsa.PublicKeyFromBytes(addFileTransaction.OwnerAddress); err != nil {
		return fmt.Errorf("transaction validation failed: OwnerAddress is not a valid ECDSA public key")
	}
	if _, err := ecdsa.PublicKeyFromBytes(addFileTransaction.NodeAddress); err != nil {
		return fmt.Errorf("transaction validation failed: NodeAddress is not a valid ECDSA public key")
	}

	if bytes.Equal(addFileTransaction.OwnerAddress, addFileTransaction.NodeAddress) {
		return fmt.Errorf("transaction validation failed: OwnerAddress and NodeAddress must be different")
	}

	// Verify that checksum is a valid SHA256 hash
	if len(addFileTransaction.Checksum) != sha256.Size {
		return fmt.Errorf("transaction validation failed: Checksum is not a valid SHA256 hash")
	}

	// Verify that NodeID is valid peer.ID
	if !isValidPeerID(addFileTransaction.NodeID) {
		return fmt.Errorf("transaction validation failed: NodeCID is not a valid CID")
	}

	// Verify that IPFSClientNodeAddrInfo and IPFSStorageNodeAddrInfo are valid peer.AddrInfo
	if addFileTransaction.IPFSClientNodeAddrInfo.ID == "" || addFileTransaction.IPFSStorageNodeAddrInfo.ID == "" {
		return fmt.Errorf("transaction validation failed: IPFSClientNodeAddrInfo and IPFSStorageNodeAddrInfo must have valid peer.IDs")
	}

	if addFileTransaction.IPFSClientNodeAddrInfo.ID == addFileTransaction.IPFSStorageNodeAddrInfo.ID {
		return fmt.Errorf("transaction validation failed: IPFSClientNodeAddrInfo and IPFSStorageNodeAddrInfo must have different peer.IDs")
	}

	if addFileTransaction.IPFSStorageNodeAddrInfo.ID == addFileTransaction.NodeID {
		return fmt.Errorf("transaction validation failed: IPFSStorageNodeAddrInfo and NodeID must have different peer.IDs")
	}

	if !isValidPeerID(addFileTransaction.IPFSClientNodeAddrInfo.ID) || !isValidPeerID(addFileTransaction.IPFSStorageNodeAddrInfo.ID) {
		return fmt.Errorf("transaction validation failed: IPFSClientNodeAddrInfo and IPFSStorageNodeAddrInfo must have valid peer.IDs")
	}

	// Verify that FileCID is valid CID
	if !isValidCID(addFileTransaction.FileCid) {
		return fmt.Errorf("transaction validation failed: FileCid is not a valid CID")
	}

	// Verify timestamps order (AnnouncementTimestamp < TransactionTimestamp)
	if !isValidTimestampOrder(addFileTransaction.AnnouncementTimestamp, addFileTransaction.TransactionTimestamp) {
		return fmt.Errorf("transaction validation failed: Timestamps are not in the correct order")
	}

	// Verify that filename and extension have been provided
	if len(addFileTransaction.Filename) == 0 || len(addFileTransaction.Extension) == 0 {
		return fmt.Errorf("transaction validation failed: Filename and extension must be provided")
	}

	// Verify the client announcement step
	clientAnnouncement := transaction.ClientAnnouncement{
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

	if err := clientAnnouncement.Verify(clientAnnouncement, clientAnnouncement.OwnerSignature, clientAnnouncement.OwnerAddress); err != nil {
		return fmt.Errorf("transaction validation failed: Client announcement signature is invalid: %s", err)
	}

	// Verify the transaction step
	if err := addFileTransaction.Verify(addFileTransaction, addFileTransaction.TransactionSignature, addFileTransaction.NodeAddress); err != nil {
		return fmt.Errorf("transaction validation failed: Transaction signature is invalid: %s", err)
	}

	return nil
}

// DeleteFileTransactionValidator for DeleteFileTransaction
type DeleteFileTransactionValidator struct{}

func (v DeleteFileTransactionValidator) Validate(tx transaction.Transaction) error {
	deleteFileTransaction, ok := tx.(*transaction.DeleteFileTransaction)
	if !ok {
		return fmt.Errorf("transaction validation failed: Transaction is not a DeleteFileTransaction")
	}

	// Verify that TransactionID is a valid UUID
	if _, err := uuid.Parse(deleteFileTransaction.TransactionID.String()); err != nil {
		return fmt.Errorf("transaction validation failed: TransactionID is not a valid UUID")
	}

	// Verify that OwnerAddress is a valid ECDSA public key
	if _, err := ecdsa.PublicKeyFromBytes(deleteFileTransaction.OwnerAddress); err != nil {
		return fmt.Errorf("transaction validation failed: OwnerAddress is not a valid ECDSA public key")
	}

	// Verify that FileCID is a valid CID
	if !isValidCID(deleteFileTransaction.FileCid) {
		return fmt.Errorf("transaction validation failed: FileCid is not a valid CID")
	}

	// Verify TransactionTimestamp
	if deleteFileTransaction.TransactionTimestamp <= 0 {
		return fmt.Errorf("transaction validation failed: TransactionTimestamp must be greater than 0")
	}

	if deleteFileTransaction.TransactionTimestamp >= time.Now().UTC().Unix() {
		return fmt.Errorf("transaction validation failed: TransactionTimestamp must be in the past")
	}

	// Verify the transaction signature
	if err := deleteFileTransaction.Verify(deleteFileTransaction, deleteFileTransaction.TransactionSignature, deleteFileTransaction.OwnerAddress); err != nil {
		return fmt.Errorf("transaction validation failed: Transaction signature is invalid: %w", err)
	}

	return nil
}
