package fileregistry

import (
	"fmt"
	"os"
	"testing"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

const (
	testRegistryFilename          = "test"
	clientECDSAPrivateKeyFilename = "private_key"
	clientECDSAPublicKeyFilename  = "public_key"
	clientECDSAStoragePath        = "."
)

func generateFakeBlock() block.Block {
	prevBlockStr := fmt.Sprintf("PrevBlock-%d", time.Now().Unix())
	merkleRootStr := fmt.Sprintf("MerkleRoot-%d", time.Now().Unix())
	minerAddrStr := fmt.Sprintf("MinerAddr-%d", time.Now().Unix())
	signatureStr := fmt.Sprintf("Signature-%d", time.Now().Unix())
	block := block.Block{
		Header: block.Header{
			Version:    1,
			PrevBlock:  []byte(prevBlockStr),
			MerkleRoot: []byte(merkleRootStr),
			TargetBits: 1,
			Timestamp:  time.Now().Unix(),
			Height:     1,
			Nonce:      1,
			MinerAddr:  []byte(minerAddrStr),
			Signature:  []byte(signatureStr),
		},
		Transactions: nil,
	}
	return block
}

func TestDefaultRegistryAddFile(t *testing.T) {
	// Mock
	clientECDSAKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		t.Fatalf("Error generating ECDSA key pair: %s", err)
	}
	err = clientECDSAKeyPair.SaveKeys(clientECDSAPrivateKeyFilename, clientECDSAPublicKeyFilename, clientECDSAStoragePath)
	if err != nil {
		t.Fatalf("Error saving ECDSA key pair: %s", err)
	}

	addFileTransaction, err := transaction.GenFakeAddTransactionWithSameOwner(clientECDSAKeyPair)
	if err != nil {
		t.Fatalf("Error generating fake AddFileTransaction: %s", err)
	}

	logger := ipfsLog.Logger("test")

	mockConfig := &config.Config{
		FileRights: 0700,
	}

	// Test
	registryManager := NewDefaultFileRegistryManager(logger, mockConfig, testRegistryFilename)

	fileRegistry, err := NewDefaultFileRegistry(logger, mockConfig, registryManager)
	if err != nil {
		t.Errorf("Error creating new file registry: %v", err)
	}

	err = fileRegistry.Add(*addFileTransaction)
	if err != nil {
		t.Errorf("Error adding file to registry: %v", err)
	}
}

func TestDefaultRegistryGetFile(t *testing.T) {
	// Mock
	clientECDSAKeyPair, err := ecdsa.LoadKeys(clientECDSAPrivateKeyFilename, clientECDSAPublicKeyFilename, clientECDSAStoragePath)
	if err != nil {
		t.Fatalf("Error loading ECDSA key pair: %s", err)
	}

	logger := ipfsLog.Logger("test")

	mockConfig := &config.Config{
		FileRights: 0700,
	}

	// Test
	registryManager := NewDefaultFileRegistryManager(logger, mockConfig, testRegistryFilename)

	fileRegistry, err := NewDefaultFileRegistry(logger, mockConfig, registryManager)
	if err != nil {
		t.Errorf("Error creating new file registry: %v", err)
	}

	clientPubKey, err := clientECDSAKeyPair.PublicKeyToBytes()
	if err != nil {
		t.Fatalf("Error converting public key to bytes: %s", err)
	}
	clientPubKeyToString := fmt.Sprintf("%x", clientPubKey)

	files := fileRegistry.Get(clientPubKeyToString)
	if len(files) == 0 {
		t.Errorf("No files found for the client")
	}
}

func TestDefaultRegistryDeleteFile(t *testing.T) {
	// Mock
	clientECDSAKeyPair, err := ecdsa.LoadKeys(clientECDSAPrivateKeyFilename, clientECDSAPublicKeyFilename, clientECDSAStoragePath)
	if err != nil {
		t.Fatalf("Error loading ECDSA key pair: %s", err)
	}

	randomFileCid := merkledag.NodeWithData(unixfs.FilePBData([]byte("fileCID"), uint64(len([]byte("fileCID"))))).Cid() // Random CIDv0
	deleteFileTransaction := transaction.NewDeleteFileTransaction(clientECDSAKeyPair, randomFileCid)

	logger := ipfsLog.Logger("test")

	mockConfig := &config.Config{
		FileRights: 0700,
	}

	// Test
	registryManager := NewDefaultFileRegistryManager(logger, mockConfig, testRegistryFilename)

	fileRegistry, err := NewDefaultFileRegistry(logger, mockConfig, registryManager)
	if err != nil {
		t.Errorf("Error creating new file registry: %v", err)
	}

	err = fileRegistry.Delete(*deleteFileTransaction)
	if err != nil {
		t.Errorf("Error deleting file from registry: %v", err)
	}
}

func TestDefaultRegistryUpdate(t *testing.T) {
	// Mock
	clientECDSAKeyPair, err := ecdsa.LoadKeys(clientECDSAPrivateKeyFilename, clientECDSAPublicKeyFilename, clientECDSAStoragePath)
	if err != nil {
		t.Fatalf("Error loading ECDSA key pair: %s", err)
	}

	logger := ipfsLog.Logger("test")

	mockConfig := &config.Config{
		FileRights: 0700,
	}

	addFileTransaction, err := transaction.GenFakeAddTransactionWithSameOwner(clientECDSAKeyPair)
	if err != nil {
		t.Fatalf("Error generating fake AddFileTransaction: %s", err)
	}

	fakeBlock1 := generateFakeBlock()
	fakeBlock1.Transactions = append(fakeBlock1.Transactions, addFileTransaction)

	randomFileCid := merkledag.NodeWithData(unixfs.FilePBData([]byte("fileCID"), uint64(len([]byte("fileCID"))))).Cid() // Random CIDv0
	deleteFileTransaction := transaction.NewDeleteFileTransaction(clientECDSAKeyPair, randomFileCid)

	fakeBlock2 := generateFakeBlock()
	fakeBlock2.Transactions = append(fakeBlock2.Transactions, deleteFileTransaction)

	// Test
	registryManager := NewDefaultFileRegistryManager(logger, mockConfig, testRegistryFilename)

	fileRegistry, err := NewDefaultFileRegistry(logger, mockConfig, registryManager)
	if err != nil {
		t.Errorf("Error creating new file registry: %v", err)
	}

	fmt.Println("Block transactions: ", fakeBlock1.Transactions)

	err = fileRegistry.UpdateFromBlock(fakeBlock1)
	if err != nil {
		t.Errorf("Error updating registry from block: %v", err)
	}

	err = fileRegistry.UpdateFromBlock(fakeBlock2)
	if err != nil {
		t.Errorf("Error updating registry from block: %v", err)
	}

	// Delete the registry file
	err = os.Remove(testRegistryFilename)
	if err != nil {
		t.Errorf("Error deleting registry file: %v", err)
	}
	err = os.Remove(clientECDSAPrivateKeyFilename + ".pem")
	if err != nil {
		t.Errorf("Error deleting private key file: %v", err)
	}
	err = os.Remove(clientECDSAPublicKeyFilename + ".pem")
	if err != nil {
		t.Errorf("Error deleting public key file: %v", err)
	}
}
