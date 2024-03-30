package poccontext

import (
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	merkledag "github.com/ipfs/boxo/ipld/merkledag"
	unixfs "github.com/ipfs/boxo/ipld/unixfs"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func GenFakeTransaction() (*transaction.AddFileTransaction, error) {
	// Set up a valid AddFileTransaction
	nodeECDSAKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to create ECDSA key pair: %s", err)
	}

	ownerECDSAKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to create ECDSA key pair: %s", err)
	}

	ownerAesKey, err := aes.NewAESKey()
	if err != nil {
		return nil, fmt.Errorf("failed to create AES key: %s", err)
	}

	randomeNodeID, err := generateRandomPeerID()
	if err != nil {
		return nil, fmt.Errorf("failed to create peer.ID: %s", err)
	}

	// Generate a random uuid for the file CID
	// randomFilename := uuid.New().String()
	randomFilename := "fileCID"
	randomFileCid := merkledag.NodeWithData(unixfs.FilePBData([]byte(randomFilename), uint64(len([]byte(randomFilename))))).Cid() // Random CIDv0

	checksum := sha256.Sum256([]byte("checksum"))
	encryptedFilename, err := ownerAesKey.EncryptData([]byte("filename"))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt filename: %s", err)
	}

	encryptedExtension, err := ownerAesKey.EncryptData([]byte("extension"))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt extension: %s", err)
	}

	randomClientAddrInfo, err := generateRandomAddrInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to create random AddrInfo: %s", err)
	}

	randomStorageAddrInfo, err := generateRandomAddrInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to create random AddrInfo: %s", err)
	}

	announcement := transaction.NewClientAnnouncement(ownerECDSAKeyPair, randomClientAddrInfo, randomFileCid, encryptedFilename, encryptedExtension, 1234, checksum[:])
	time.Sleep(1 * time.Second)
	addFileTransaction := transaction.NewAddFileTransaction(announcement, randomFileCid, false, nodeECDSAKeyPair, randomeNodeID, randomStorageAddrInfo)
	bd, _ := addFileTransaction.Serialize()
	log.Printf("Transaction: %s\n", string(bd))

	if !consensus.ValidateTransaction(addFileTransaction) {
		return nil, fmt.Errorf("ValidateTransaction failed for a valid AddFileTransaction")
	}

	return addFileTransaction, nil
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
