package pebble

import (
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
)

// BlockDatabase defines an interface for interacing with the transaction database.
type BlockDatabase interface {
	SaveBlock(key []byte, b block.Block) error
	GetBlock(key []byte) (block.Block, error)
	VerifyBlockchainIntegrity(lastestBlock *block.Block) (bool, error)
	GetLastBlock() block.Block
	Close() error
}

// PebbleDB implements the BlockDatabase interface using the Pebble library.
type PebbleDB struct {
	db *pebble.DB
}

// NewBlockchainDB creates a new instance of PebbleDB.
func NewBlockchainDB(dbPath string) (*PebbleDB, error) {
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble database : %v", err)
	}
	return &PebbleDB{db: db}, nil
}

// SaveBlock stores a block in the database.
func (pdb *PebbleDB) SaveBlock(key []byte, b *block.Block) error {
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

// GetBlock retrives a block from the database using its key.
func (pdb *PebbleDB) GetBlock(key []byte) (*block.Block, error) {
	blockBytes, closer, err := pdb.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			// The block does not exist in the database it returns nils
			return nil, nil
		}
		return nil, fmt.Errorf("error retrieving block from the database: %v", err)
	}
	defer closer.Close()

	b, err := block.DeserializeBlock(blockBytes)
	if err != nil {
		return nil, fmt.Errorf("error deserializing block: %v", err)
	}

	return b, nil
}

// VerifyBlockchainIntegrity checks the integrity of the blockchain starting from the latest added block.
func (pdb *PebbleDB) VerifyBlockchainIntegrity(lastestBlock *block.Block) (bool, error) {
	currentBlockKey := lastestBlock.PrevBlock

	for currentBlockKey != nil {
		currentBlock, err := pdb.GetBlock(currentBlockKey)
		if err != nil {
			return false, fmt.Errorf("error retrieving block from the database: %v", err)
		}

		currentPrevBlock, err := pdb.GetBlock(currentBlock.PrevBlock)
		if err != nil {
			return false, fmt.Errorf("previous block not found")
		}

		if !consensus.ValidateBlock(currentBlock, currentPrevBlock) {
			return false, fmt.Errorf("block validation failed")
		}

		currentBlockKey = currentBlock.PrevBlock
	}
	fmt.Println("Blockchain integrity verified")
	return true, nil
}

// saveLastBlock updates the reference to the last block in the blockchain.
func (pdb *PebbleDB) saveLastBlock(lastBlock *block.Block) error {
	lastBlockBytes, err := lastBlock.Serialize()
	if err != nil {
		return fmt.Errorf("error serializing block: %v", err)
	}

	err = pdb.db.Set([]byte("lastBlockKey"), lastBlockBytes, pebble.Sync)
	if err != nil {
		return fmt.Errorf("error saving block to the database : %v", err)
	}

	return nil
}

// GetLastBlock retrieves the most recently added block from the database.
func (pdb *PebbleDB) GetLastBlock() *block.Block {
	lastBlock, err := pdb.GetBlock([]byte("lastBlockKey"))
	if err != nil {
		return nil
	}
	return lastBlock
}

// Close closes the database connection.
func (pdb *PebbleDB) Close() error {
	return pdb.db.Close()
}
