package block

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
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
func NewBlock(log *ipfsLog.ZapEventLogger, transactions []transaction.Transaction, prevBlockHash []byte, height uint32, minerAddr ecdsa.KeyPair) *Block {
	minerAddrBytes, err := minerAddr.PublicKeyToBytes()
	if err != nil {
		log.Errorln("Error converting public key to bytes")
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
	block.Header.MerkleRoot = block.ComputeMerkleRoot(log)

	log.Debugln("New block created")
	return block
}

// calculateMerkleRoot computes the Merkle root of the transactions in the block
func (b *Block) ComputeMerkleRoot(log *ipfsLog.ZapEventLogger) []byte {
	var txHashes [][]byte
	for _, tx := range b.Transactions {
		log.Debugln("Computing hash for transaction")
		txBytes, _ := json.Marshal(tx)
		txHash := sha256.Sum256(txBytes)
		txHashes = append(txHashes, txHash[:])
	}

	log.Debugln("Computing Merkle root for block")
	return computeMerkleRootForHashes(log, txHashes)
}

// calculateMerkleRootForHashes recursively calculates the Merkle root for a slice of transaction hashes
func computeMerkleRootForHashes(log *ipfsLog.ZapEventLogger, hashes [][]byte) []byte {
	if len(hashes) == 0 {
		// Handle the case where there are no transactions
		log.Debugln("No transactions to compute Merkle root")
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
	return computeMerkleRootForHashes(log, newLayer)
}

// Serialize converts the block into a byte slice
func (b *Block) Serialize(log *ipfsLog.ZapEventLogger) ([]byte, error) {
	aux := struct {
		Header       Header                           `json:"header"`
		Transactions []transaction.TransactionWrapper `json:"transactions"`
	}{
		Header: b.Header,
	}

	for _, tx := range b.Transactions {
		var txWrapped transaction.TransactionWrapper
		switch txType := tx.(type) {
		case *transaction.AddFileTransaction:
			log.Debugln("Serializing AddFileTransaction")
			data, err := json.Marshal(tx)
			if err != nil {
				log.Errorln("Error serializing AddFileTransaction")
				return nil, err
			}
			txWrapped = transaction.TransactionWrapper{
				Type: reflect.TypeOf(tx).Elem().Name(),
				Data: data,
			}
		case *transaction.DeleteFileTransaction:
			log.Debugln("Serializing DeleteFileTransaction")
			data, err := json.Marshal(tx)
			if err != nil {
				log.Errorln("Error serializing DeleteFileTransaction")
				return nil, err
			}
			txWrapped = transaction.TransactionWrapper{
				Type: reflect.TypeOf(tx).Elem().Name(),
				Data: data,
			}
		default:
			log.Errorln("Unknown transaction type")
			return nil, fmt.Errorf("unknown transaction type: %v", txType)
		}
		aux.Transactions = append(aux.Transactions, txWrapped)
	}
	return json.Marshal(aux)
}

// DeserializeBlock converts a byte slice back into a Block
func DeserializeBlock(log *ipfsLog.ZapEventLogger, data []byte) (*Block, error) {
	var aux struct {
		Header       Header                           `json:"header"`
		Transactions []transaction.TransactionWrapper `json:"transactions"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		log.Errorln("Error deserializing block")
		return nil, err
	}

	block := &Block{
		Header: aux.Header,
	}

	for _, wrappedTrx := range aux.Transactions {
		var tx transaction.Transaction
		switch wrappedTrx.Type {
		case "AddFileTransaction":
			log.Debugln("Deserializing AddFileTransaction")
			tx = &transaction.AddFileTransaction{}
		case "DeleteFileTransaction":
			log.Debugln("Deserializing DeleteFileTransaction")
			tx = &transaction.DeleteFileTransaction{}
		default:
			log.Errorln("Unknown transaction type")
			return nil, fmt.Errorf("unknown transaction type: %v", wrappedTrx.Type)
		}
		if err := json.Unmarshal(wrappedTrx.Data, &tx); err != nil {
			log.Errorln("Error deserializing transaction")
			return nil, err
		}
		block.Transactions = append(block.Transactions, tx)
	}
	return block, nil
}

// SignBlock signs the block with the given private key and adds the signature to the block header
func (b *Block) SignBlock(log *ipfsLog.ZapEventLogger, privateKey ecdsa.KeyPair) error {
	headerHash := ComputeHash(log, b)
	signature, err := privateKey.Sign(headerHash)
	if err != nil {
		log.Errorln("Error signing block")
		return err
	}

	log.Debugln("Block signed")
	b.Header.Signature = signature
	return nil
}

// IsGenesisBlock checks if the block is the genesis block
func IsGenesisBlock(log *ipfsLog.ZapEventLogger, b *Block) bool {
	if b.PrevBlock == nil && b.Header.Height == 1 {
		log.Debugln("Block is the genesis block")
		return true
	}
	log.Debugln("Block is not the genesis block")
	return false
}
