package transaction

import (
	"crypto/sha256"
	"fmt"
	"time"

	merkledag "github.com/ipfs/boxo/ipld/merkledag"
	unixfs "github.com/ipfs/boxo/ipld/unixfs"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/pkg/aes"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func GenFakeAddTransaction(log *ipfsLog.ZapEventLogger) (*AddFileTransaction, error) {
	// Set up a valid AddFileTransaction
	nodeECDSAKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		log.Errorf("failed to create ECDSA key pair: %s\n", err)
		return nil, fmt.Errorf("failed to create ECDSA key pair: %s", err)
	}

	ownerECDSAKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		log.Errorf("failed to create ECDSA key pair: %s\n", err)
		return nil, fmt.Errorf("failed to create ECDSA key pair: %s", err)
	}

	ownerAesKey, err := aes.NewAESKey()
	if err != nil {
		log.Errorf("failed to create AES key: %s\n", err)
		return nil, fmt.Errorf("failed to create AES key: %s", err)
	}

	randomeNodeID, err := genRandomPeerID(log)
	if err != nil {
		log.Errorf("failed to create peer.ID: %s\n", err)
		return nil, fmt.Errorf("failed to create peer.ID: %s", err)
	}

	randomFileCid := merkledag.NodeWithData(unixfs.FilePBData([]byte("fileCID"), uint64(len([]byte("fileCID"))))).Cid() // Random CIDv0

	checksum := sha256.Sum256([]byte("checksum"))
	encryptedFilename, err := ownerAesKey.EncryptData([]byte("filename"))
	if err != nil {
		log.Errorf("failed to encrypt filename: %s\n", err)
		return nil, fmt.Errorf("failed to encrypt filename: %s", err)
	}

	encryptedExtension, err := ownerAesKey.EncryptData([]byte("extension"))
	if err != nil {
		log.Errorf("failed to encrypt extension: %s\n", err)
		return nil, fmt.Errorf("failed to encrypt extension: %s", err)
	}

	randomClientAddrInfo, err := generateRandomAddrInfo(log)
	if err != nil {
		log.Errorf("failed to create random AddrInfo: %s\n", err)
		return nil, fmt.Errorf("failed to create random AddrInfo: %s", err)
	}

	randomStorageAddrInfo, err := generateRandomAddrInfo(log)
	if err != nil {
		log.Errorf("failed to create random AddrInfo: %s\n", err)
		return nil, fmt.Errorf("failed to create random AddrInfo: %s", err)
	}

	announcement := NewClientAnnouncement(ownerECDSAKeyPair, randomClientAddrInfo, randomFileCid, encryptedFilename, encryptedExtension, 1234, checksum[:])
	time.Sleep(1 * time.Second)
	addFileTransaction := NewAddFileTransaction(announcement, randomFileCid, false, nodeECDSAKeyPair, randomeNodeID, randomStorageAddrInfo)
	bd, _ := addFileTransaction.Serialize()
	log.Debugln("Transaction: %s\n", string(bd))

	// if !ValidateTransaction(addFileTransaction) {
	// 	return nil, fmt.Errorf("ValidateTransaction failed for a valid AddFileTransaction")
	// }

	return addFileTransaction, nil
}

func GenFakeDeleteTransaction(log *ipfsLog.ZapEventLogger) (*DeleteFileTransaction, error) {
	// Set up a valid DeleteFileTransaction
	ownerECDSAKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		log.Errorf("failed to create ECDSA key pair: %s\n", err)
		return nil, fmt.Errorf("failed to create ECDSA key pair: %s", err)
	}

	randomeFileCID := merkledag.NodeWithData(unixfs.FilePBData([]byte("fileCID"), uint64(len([]byte("fileCID"))))).Cid() // Random CIDv0

	deleteFileTransaction := NewDeleteFileTransaction(ownerECDSAKeyPair, randomeFileCID)
	time.Sleep(1 * time.Second)

	log.Debugln("Fake delete transaction created successfully")
	return deleteFileTransaction, nil
}

func genRandomPeerID(log *ipfsLog.ZapEventLogger) (peer.ID, error) {
	// Generate a new RSA key pair for this host
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		log.Errorln("Error generating RSA key pair")
		return "", err
	}

	// Convert the RSA key pair into a libp2p Peer ID
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		log.Errorln("Error converting RSA key pair to Peer ID")
		return "", err
	}

	log.Debugln("Random Peer ID generated successfully")
	return pid, nil
}

func generateRandomAddrInfo(log *ipfsLog.ZapEventLogger) (peer.AddrInfo, error) {
	// Generate a new RSA key pair for this host
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		log.Errorln("Error generating RSA key pair")
		return peer.AddrInfo{}, err
	}

	// Convert the RSA key pair into a libp2p Peer ID
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		log.Errorln("Error converting RSA key pair to Peer ID")
		return peer.AddrInfo{}, err
	}

	log.Debugln("Random AddrInfo generated successfully")
	return peer.AddrInfo{ID: pid}, nil
}
