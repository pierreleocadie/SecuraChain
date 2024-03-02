package pebble

import (
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
)

// BlockDatabase defines an interface for interacing with the transaction database.
type BlockDatabase interface {
	SaveBlock(key []byte, bx block.Block) error
	GetBlock(key []byte) (block.Block, error)
	VerifyBlockchainIntegrity(lastestBlockAdded *block.Block) (bool, error)
	// IsIn(block *block.Block) (bool, error)
	GetLastBlock() block.Block
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
func (pdb *PebbleTransactionDB) SaveBlock(key []byte, b *block.Block) error {
	bxBytes, err := b.Serialize()
	if err != nil {
		return fmt.Errorf("error serializing block: %v", err)
	}

	err = pdb.db.Set(key, bxBytes, pebble.Sync)
	if err != nil {
		return fmt.Errorf("error saving block to the database : %v", err)
	}

	err = pdb.saveLastBlock(b)
	return nil
}

// GetBlock retrives a block from the database.
func (pdb *PebbleTransactionDB) GetBlock(key []byte) (*block.Block, error) {
	bxBytes, closer, err := pdb.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			// The block does not exist in the database it returns nils
			return nil, nil
		}
		return nil, fmt.Errorf("error retrieving block from the database: %v", err)
	}
	defer closer.Close()

	blockDeserialize, err := block.DeserializeBlock(bxBytes)
	if err != nil {
		return nil, fmt.Errorf("error deserializing block: %v", err)
	}

	return blockDeserialize, nil
}

// VerifyBlockchainIntegrity checks the blockchain for integrity.
func (pdb *PebbleTransactionDB) VerifyBlockchainIntegrity(lastestBlockAdded *block.Block) (bool, error) {
	currentBlockKey := lastestBlockAdded.PrevBlock

	for {
		if currentBlockKey == nil {
			// Genesis block reached
			fmt.Println("Blockchain integrity verified")
			return true, nil
		}

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
}

// // IsIn checks if a blocks is in the blockchain
// func (pdb *PebbleTransactionDB) IsIn(b *block.Block) (bool, error) {
// 	key := block.ComputeHash(b)

// 	_, closer, err := pdb.db.Get(key)
// 	if err != nil {
// 		if err == pebble.ErrNotFound {
// 			// The block does not exist in the database.
// 			return false, nil
// 		}
// 		return false, fmt.Errorf("error retrieving block from database: %v", err)
// 	}

// 	if closer != nil {
// 		closer.Close()
// 	}

// 	// The block exists in the database.
// 	return true, nil
// }

// saveLastBlock saves the last block to the PebbleTransactionDB.
// It serializes the lastBlock object into JSON format and stores it in the database.
// The last block is saved with the key "lastKey".
// Returns an error if there is an issue serializing the block or saving it to the database.
func (pdb *PebbleTransactionDB) saveLastBlock(lastBlock *block.Block) error {
	lastBlockBytes, err := lastBlock.Serialize()
	if err != nil {
		return fmt.Errorf("error serializing block: %v", err)
	}

	err = pdb.db.Set([]byte("lastKey"), lastBlockBytes, pebble.Sync)
	if err != nil {
		return fmt.Errorf("error saving block to the database : %v", err)
	}

	return nil
}

// GetLastBlock retrieves the last block from the database.
// It returns the last block and an error if any occurred.
func (pdb *PebbleTransactionDB) GetLastBlock() *block.Block {
	lastBlock, err := pdb.GetBlock([]byte("lastKey"))
	if err != nil {
		return nil
	}

	return lastBlock
}

// Close closes the database connection.
func (pdb *PebbleTransactionDB) Close() error {
	return pdb.db.Close()
}
