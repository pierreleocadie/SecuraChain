package consensus

import (
	"crypto/sha256"
	"testing"
	"time"

	merkledag "github.com/ipfs/boxo/ipld/merkledag"
	unixfs "github.com/ipfs/boxo/ipld/unixfs"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func TestAddFileTransactionValidator_ValidTransaction(t *testing.T) {
	t.Parallel()

	// Set up a valid AddFileTransaction
	nodeECDSAKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		t.Errorf("Failed to create ECDSA key pair: %s", err)
	}

	ownerECDSAKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		t.Errorf("Failed to create ECDSA key pair: %s", err)
	}

	ownerAesKey, err := aes.NewAESKey()
	if err != nil {
		t.Errorf("Failed to create AES key: %s", err)
	}

	randomeNodeCID := merkledag.NodeWithData(unixfs.FilePBData([]byte("nodeCID"), uint64(len([]byte("nodeCID"))))).Cid() // Random CIDv0
	randomeFileCID := merkledag.NodeWithData(unixfs.FilePBData([]byte("fileCID"), uint64(len([]byte("fileCID"))))).Cid() // Random CIDv0

	checksum := sha256.Sum256([]byte("checksum"))
	encryptedFilename, err := ownerAesKey.EncryptData([]byte("filename"))
	if err != nil {
		t.Errorf("Failed to encrypt filename: %s", err)
	}

	encryptedExtension, err := ownerAesKey.EncryptData([]byte("extension"))
	if err != nil {
		t.Errorf("Failed to encrypt extension: %s", err)
	}

	announcement := transaction.NewClientAnnouncement(ownerECDSAKeyPair, encryptedFilename, encryptedExtension, 1234, checksum[:])
	time.Sleep(1 * time.Second)
	response := transaction.NewStorageNodeResponse(nodeECDSAKeyPair, randomeNodeCID, "", announcement)
	time.Sleep(1 * time.Second)
	fileTransfer := transaction.NewFileTransferHTTPRequest(announcement, response, nil, ownerECDSAKeyPair)
	time.Sleep(1 * time.Second)
	addFileTransaction := transaction.NewAddFileTransaction(announcement, response, fileTransfer, randomeFileCID, false, ownerECDSAKeyPair)

	if !ValidateTransaction(addFileTransaction) {
		t.Errorf("ValidateTransaction failed for a valid AddFileTransaction")
	}
}

func TestDeleteFileTransactionValidator_ValidTransaction(t *testing.T) {
	t.Parallel()

	// Set up a valid DeleteFileTransaction
	ownerECDSAKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		t.Errorf("Failed to create ECDSA key pair: %s", err)
	}

	randomeFileCID := merkledag.NodeWithData(unixfs.FilePBData([]byte("fileCID"), uint64(len([]byte("fileCID"))))).Cid() // Random CIDv0

	deleteFileTransaction := transaction.NewDeleteFileTransaction(ownerECDSAKeyPair, randomeFileCID)
	time.Sleep(1 * time.Second)

	if !ValidateTransaction(deleteFileTransaction) {
		t.Errorf("ValidateTransaction failed for a valid DeleteFileTransaction")
	}
}
