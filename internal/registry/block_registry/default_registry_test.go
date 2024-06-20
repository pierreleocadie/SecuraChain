package blockregistry

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	merkledag "github.com/ipfs/boxo/ipld/merkledag"
	unixfs "github.com/ipfs/boxo/ipld/unixfs"
	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

const registryName = "test"

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

func generateFakeCID() path.ImmutablePath {
	randomUUID := uuid.New().String()
	randomCID := fmt.Sprintf("CID-%d", time.Now().Unix())
	fakeCid := path.FromCid(merkledag.NodeWithData(unixfs.FilePBData([]byte(randomUUID), uint64(len([]byte(randomCID))))).Cid())
	return fakeCid
}

func TestDefaultBlockRegistry(t *testing.T) {
	// Create a mock logger
	logger := ipfsLog.Logger("test")

	// Create a mock config
	mockConfig := &config.Config{
		FileRights: 0700,
	}

	// Create a block registry manager
	blockRegistryManager := NewDefaultBlockRegistryManager(logger, mockConfig, registryName)

	// Create a new block registry
	blockRegistry, err := NewDefaultBlockRegistry(logger, mockConfig, blockRegistryManager)
	if err != nil {
		t.Errorf("Error creating new block registry: %v", err)
	}

	// Add the block to the registry
	fakePeerAddrInfo1, err := generateRandomAddrInfo()
	if err != nil {
		t.Errorf("Error generating random AddrInfo: %v", err)
	}

	err = blockRegistry.Add(generateFakeBlock(), generateFakeCID(), fakePeerAddrInfo1)
	if err != nil {
		t.Errorf("Error adding block to registry: %v", err)
	}

	// Add the block to the registry
	fakePeerAddrInfo2, err := generateRandomAddrInfo()
	if err != nil {
		t.Errorf("Error generating random AddrInfo: %v", err)
	}

	err = blockRegistry.Add(generateFakeBlock(), generateFakeCID(), fakePeerAddrInfo2)
	if err != nil {
		t.Errorf("Error adding block to registry: %v", err)
	}
}

func TestDefaultRegistryLoading(t *testing.T) {
	// Create a mock logger
	logger := ipfsLog.Logger("test")

	// Create a mock config
	mockConfig := &config.Config{
		FileRights: 0700,
	}

	// Create a block registry manager
	blockRegistryManager := NewDefaultBlockRegistryManager(logger, mockConfig, registryName)

	// Create a new block registry
	blockRegistry, err := NewDefaultBlockRegistry(logger, mockConfig, blockRegistryManager)
	if err != nil {
		t.Errorf("Error creating new block registry: %v", err)
	}

	// Add the block to the registry
	fakePeerAddrInfo1, err := generateRandomAddrInfo()
	if err != nil {
		t.Errorf("Error generating random AddrInfo: %v", err)
	}

	err = blockRegistry.Add(generateFakeBlock(), generateFakeCID(), fakePeerAddrInfo1)
	if err != nil {
		t.Errorf("Error adding block to registry: %v", err)
	}

	// Delete the registry file
	err = os.Remove(registryName)
	if err != nil {
		t.Errorf("Error deleting registry file: %v", err)
	}
}
