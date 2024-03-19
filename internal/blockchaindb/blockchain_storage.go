package blockchaindb

import (
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
)

// BlockchainStorage defines an interface for interacting with the blockchain storage.
type BlockchainStorage interface {
	SaveBlock(key []byte, b *block.Block) error
	GetBlock(key []byte) (*block.Block, error)
	VerifyIntegrity() (bool, error)
	GetLastBlock() *block.Block
	Close() error
}

// BlockchainDB wraps a Pebble database instance to store blockchain data.
type BlockchainDB struct {
	db *pebble.DB
}

// BlockchainDB wraps a Pebble database instance to store blockchain data.
func NewBlockchainDB(dbPath string) (*BlockchainDB, error) {
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble database : %v", err)
	}
	return &BlockchainDB{db: db}, nil
}

// SaveBlock serializes and stores a given block in the database.
func (pdb *BlockchainDB) SaveBlock(key []byte, b *block.Block) error {
	blockBytes, err := b.Serialize()
	if err != nil {
		return fmt.Errorf("error serializing block: %v", err)
	}

	err = pdb.db.Set(key, blockBytes, pebble.Sync)
	if err != nil {
		return fmt.Errorf("error saving block to the database : %v", err)
	}

	return pdb.saveLastBlock(b)
}

// GetBlock retrieves a block from the database using its key.
func (pdb *BlockchainDB) GetBlock(key []byte) (*block.Block, error) {
	blockData, closer, err := pdb.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil // Block not exists
		}
		return nil, fmt.Errorf("error retrieving block : %v", err)
	}
	defer closer.Close()

	b, err := block.DeserializeBlock(blockData)
	if err != nil {
		return nil, fmt.Errorf("error deserializing block: %v", err)
	}

	return b, nil
}

// VerifyIntegrity verifies the integrity of the blockchain stored in the BlockchainDB based on the last block.
func (pdb *BlockchainDB) VerifyIntegrity() (bool, error) {

	for blockKey := pdb.GetLastBlock().PrevBlock; blockKey != nil; {
		currentBlock, err := pdb.GetBlock(blockKey)
		if err != nil {
			return false, fmt.Errorf("error retrieving block: %v", err)
		}

		prevBlock, err := pdb.GetBlock(currentBlock.PrevBlock)
		if err != nil {
			return false, fmt.Errorf("previous block not found")
		}

		if !consensus.ValidateBlock(currentBlock, prevBlock) {
			return false, fmt.Errorf("block validation failed")
		}

		blockKey = currentBlock.PrevBlock
	}
	return true, nil
}

// saveLastBlock updates the reference to the last block in the blockchain.
func (pdb *BlockchainDB) saveLastBlock(lastBlock *block.Block) error {
	serializedBlock, err := lastBlock.Serialize()
	if err != nil {
		return fmt.Errorf("error serializing block: %v", err)
	}

	if err := pdb.db.Set([]byte("lastBlockKey"), serializedBlock, pebble.Sync); err != nil {
		return fmt.Errorf("error updating last block reference: %v", err)
	}
	return nil
}

// GetLastBlock retrieves the most recently added block from the database.
func (pdb *BlockchainDB) GetLastBlock() *block.Block {
	lastBlock, err := pdb.GetBlock([]byte("lastBlockKey"))
	if err != nil {
		return nil
	}
	return lastBlock
}

// Close closes the database connection.
func (pdb *BlockchainDB) Close() error {
	return pdb.db.Close()
}
