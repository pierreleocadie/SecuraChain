package pebble

import (
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// BlockDatabase defines an interface for interacing with the transaction database.
type BlockDatabase interface {
	SaveBlock(key []byte, bx block.Block) error
	GetBlock(key []byte) (block.Block, error)
	Close() error
}

// PebbleTransactionDB implements the BlockDatabase interface using the Pebble library.
type PebbleTransactionDB struct {
	db *pebble.DB
}

// NewPebbleTransactionDB creates a new instance of PebbleTransactionDB.
func NewPebbleTransactionDB(dbPath string) (*PebbleTransactionDB, error) {
	options := &pebble.Options{}
	db, err := pebble.Open(dbPath, options)
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble database : %v", err)
	}
	return &PebbleTransactionDB{db: db}, nil
}

// SaveBlock stores a block in the database.
func (pdb *PebbleTransactionDB) SaveBlock(key []byte, bx *block.Block) error {
	bxBytes, err := json.Marshal(bx)
	if err != nil {
		return fmt.Errorf("error serializing block: %v", err)
	}

	err = pdb.db.Set(key, bxBytes, pebble.Sync)
	if err != nil {
		return fmt.Errorf("error saving block to the database : %v", err)
	}
	return nil
}

// GetBlock retrives a block from the database.
func (pdb *PebbleTransactionDB) GetBlock(key []byte) (*block.Block, error) {
	bxBytes, closer, err := pdb.db.Get(key)
	if err != nil {
		return nil, fmt.Errorf("error retrieving block from the database: %v", err)
	}
	defer closer.Close()

	var blockDeserialize *block.Block
	err = json.Unmarshal(bxBytes, &blockDeserialize)
	if err != nil {
		return nil, fmt.Errorf("error deserializing block: %v", err)
	}
	return blockDeserialize, nil
}

// VerifyBlockchainIntegrity checks the blockchain for integrity.
func (pdb *PebbleTransactionDB) VerifyBlockchainIntegrity(lastestBlockAdded *block.Block) (bool, error) {
	latestBlockKey := block.ComputeHash(lastestBlockAdded)

	currentBlockKey := lastestBlockKey
	for {
		if currentBlockKey == nil{
			// Genesis block reached
			fmt.Println("Blockchain integrity verified")
			return true, nil
		}

		currentBlock, err := pdb.GetBlock(currentBlockKey)
		if err != nil {
			return false, fmt.Errorf("error retrieving block from the database: %v", err)
		}
		prevBlock, err = block.DeserializeBlock(currentBlock.PrevBlock)
		if err != nil {
			return false, fmt.Errorf("error deserializing previous block: %v", err)
		}
		currentBlockKey = block.ComputeHash(prevBlock)
}

// Close closes the database connection.
func (pdb *PebbleTransactionDB) Close() error {
	return pdb.db.Close()
}
