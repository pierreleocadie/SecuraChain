package block

import (
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

const (
	version    = 1
	targetBits = 24
)

type Header struct {
	Version    uint32 `json:"version"`     // Blockchain version
	PrevBlock  []byte `json:"prev_block"`  // 256-bit hash of the previous block header
	MerkleRoot []byte `json:"merkle_root"` // 256-bit hash based on all of the transactions in the block
	TargetBits uint32 `json:"target_bits"` // Target bits is a way to set the mining difficulty
	Timestamp  int64  `json:"timestamp"`   // Current timestamp as seconds since 1970-01-01T00:00 UTC
	Height     uint32 `json:"height"`      // Block height
	Nonce      uint32 `json:"nonce"`       // 32-bit number (starts at 0) used to generate the required hash
	MinerAddr  []byte `json:"miner_addr"`  // ECDSA public key of the miner
	Signature  []byte `json:"signature"`   // ECDSA signature of the block header
}

type Block struct {
	Header
	Transactions []transaction.Transaction `json:"transactions"`
}

// NewBlock creates a new block using the provided transactions and the previous block hash
func NewBlock(transactions []transaction.Transaction, prevBlockHash []byte, height uint32, minerAddr ecdsa.KeyPair) *Block {
	minerAddrBytes, err := minerAddr.PublicKeyToBytes()
	if err != nil {
		return nil
	}

	block := &Block{
		Header: Header{
			Version:    version,
			PrevBlock:  prevBlockHash,
			TargetBits: targetBits,
			Timestamp:  time.Now().Unix(),
			Height:     height,
			MinerAddr:  minerAddrBytes,
		},
		Transactions: transactions,
	}
	block.Header.MerkleRoot = block.ComputeMerkleRoot()
	return block
}

// calculateMerkleRoot computes the Merkle root of the transactions in the block
func (b *Block) ComputeMerkleRoot() []byte {
	var txHashes [][]byte
	for _, tx := range b.Transactions {
		txBytes, _ := json.Marshal(tx)
		txHash := sha256.Sum256(txBytes)
		txHashes = append(txHashes, txHash[:])
	}
	return computeMerkleRootForHashes(txHashes)
}

// calculateMerkleRootForHashes recursively calculates the Merkle root for a slice of transaction hashes
func computeMerkleRootForHashes(hashes [][]byte) []byte {
	if len(hashes) == 0 {
		// Handle the case where there are no transactions
		return []byte{}
	}

	if len(hashes) == 1 {
		return hashes[0]
	}

	var newLayer [][]byte
	for i := 0; i < len(hashes); i += 2 {
		if i+1 < len(hashes) {
			combinedHash := sha256.Sum256(append(hashes[i], hashes[i+1]...))
			newLayer = append(newLayer, combinedHash[:])
		} else {
			newLayer = append(newLayer, hashes[i])
		}
	}
	return computeMerkleRootForHashes(newLayer)
}

// Serialize converts the block into a byte slice
func (b *Block) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

// DeserializeBlock converts a byte slice back into a Block
func DeserializeBlock(data []byte) (*Block, error) {
	var block Block
	err := json.Unmarshal(data, &block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

// SignBlock signs the block with the given private key and adds the signature to the block header
func (b *Block) SignBlock(privateKey ecdsa.KeyPair) error {
	headerHash := ComputeHash(b)
	signature, err := privateKey.Sign(headerHash)
	if err != nil {
		return err
	}

	b.Header.Signature = signature
	return nil
}
