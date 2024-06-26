package consensus

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	merkledag "github.com/ipfs/boxo/ipld/merkledag"
	unixfs "github.com/ipfs/boxo/ipld/unixfs"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func TestAddFileTransactionValidator_ValidTransaction(t *testing.T) {
	t.Parallel()

	// Set up a valid AddFileTransaction
	nodeECDSAKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		t.Errorf("failed to create ECDSA key pair: %s", err)
	}

	ownerECDSAKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		t.Errorf("failed to create ECDSA key pair: %s", err)
	}

	ownerAesKey, err := aes.NewAESKey()
	if err != nil {
		t.Errorf("failed to create AES key: %s", err)
	}

	randomeNodeID, err := generateRandomPeerID()
	if err != nil {
		t.Errorf("failed to create peer.ID: %s", err)
	}

	randomFileCid := merkledag.NodeWithData(unixfs.FilePBData([]byte("fileCID"), uint64(len([]byte("fileCID"))))).Cid() // Random CIDv0

	checksum := sha256.Sum256([]byte("checksum"))
	encryptedFilename, err := ownerAesKey.EncryptData([]byte("filename"))
	if err != nil {
		t.Errorf("failed to encrypt filename: %s", err)
	}

	encryptedExtension, err := ownerAesKey.EncryptData([]byte("extension"))
	if err != nil {
		t.Errorf("failed to encrypt extension: %s", err)
	}

	randomClientAddrInfo, err := generateRandomAddrInfo()
	if err != nil {
		t.Errorf("failed to create random AddrInfo: %s", err)
	}

	randomStorageAddrInfo, err := generateRandomAddrInfo()
	if err != nil {
		t.Errorf("failed to create random AddrInfo: %s", err)
	}

	announcement := transaction.NewClientAnnouncement(ownerECDSAKeyPair, randomClientAddrInfo, randomFileCid, encryptedFilename, encryptedExtension, 1234, checksum[:])
	time.Sleep(1 * time.Second)
	addFileTransaction := transaction.NewAddFileTransaction(*announcement, randomFileCid, nodeECDSAKeyPair, randomeNodeID, randomStorageAddrInfo)
	ba, _ := announcement.Serialize()
	t.Log(string(ba))
	bd, err := addFileTransaction.Serialize()
	if err != nil {
		t.Errorf("failed to serialize AddFileTransaction: %s", err)
	}
	t.Log(string(bd))

	txValidatorFactory := DefaultTransactionValidatorFactory{}
	txValidator, err := txValidatorFactory.GetValidator(addFileTransaction)
	if err != nil {
		t.Errorf("failed to get AddFileTransaction validator: %s", err)
	}
	if err := txValidator.Validate(addFileTransaction); err != nil {
		fmt.Printf("Error: %v\n", err)
		t.Errorf("validateTransaction failed for a valid AddFileTransaction")
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

	txValidatorFactory := DefaultTransactionValidatorFactory{}
	txValidator, err := txValidatorFactory.GetValidator(deleteFileTransaction)
	if err != nil {
		t.Errorf("Failed to get AddFileTransaction validator: %s", err)
	}
	if err := txValidator.Validate(deleteFileTransaction); err != nil {
		t.Errorf("ValidateTransaction failed for a valid DeleteFileTransaction")
	}
}

func generateRandomPeerID() (peer.ID, error) {
	// Generate a new RSA key pair for this host
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		return "", err
	}

	// Convert the RSA key pair into a libp2p Peer ID
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return "", err
	}

	return pid, nil
}

func generateRandomAddrInfo() (peer.AddrInfo, error) {
	// Generate a new RSA key pair for this host
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		return peer.AddrInfo{}, err
	}

	// Convert the RSA key pair into a libp2p Peer ID
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return peer.AddrInfo{}, err
	}

	return peer.AddrInfo{ID: pid}, nil
}
