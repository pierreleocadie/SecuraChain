package blockchaindb

import (
	"fmt"

	"github.com/cockroachdb/pebble"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
)

// PebbleDB wraps a Pebble database instance to store blockchain data.
type PebbleDB struct {
	db             *pebble.DB
	log            *ipfsLog.ZapEventLogger
	blockValidator consensus.BlockValidator
}

// NewPebbleDB wraps a Pebble database instance to store blockchain data.
func NewPebbleDB(logger *ipfsLog.ZapEventLogger, dbPath string, bValidator consensus.BlockValidator) (*PebbleDB, error) {
	db, err := pebble.Open(dbPath,
		&pebble.Options{
			Logger: logger,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble database : %v", err)
	}
	logger.Info("Pebble database opened successfully")
	return &PebbleDB{
		db:             db,
		log:            logger,
		blockValidator: bValidator,
	}, nil
}

// AddBlockToBlockchain adds a block to the blockchain database.
func (pdb *PebbleDB) AddBlock(b block.Block) error {
	if b.IsGenesisBlock() {
		return pdb.addGenesisBlock(b)
	}
	return pdb.addNormalBlock(b)
}

// GetBlock retrieves a block from the database using its key.
func (pdb *PebbleDB) GetBlock(key []byte) (block.Block, error) {
	blockData, closer, err := pdb.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			pdb.log.Warn("block not exists in the database")
			return block.Block{}, pebble.ErrNotFound
		}
		return block.Block{}, fmt.Errorf("error retrieving block : %v", err)
	}
	defer closer.Close()

	b, err := block.DeserializeBlock(blockData)
	if err != nil {
		return block.Block{}, fmt.Errorf("error deserializing block: %v", err)
	}

	pdb.log.Debugln("block retrieved successfully")
	return b, nil
}

// VerifyIntegrity verifies the integrity of the blockchain stored in the BlockchainDB based on the last block.
func (pdb *PebbleDB) VerifyIntegrity() error {
	pdb.log.Infoln("Verifying blockchain integrity")

	// Retrieve the last block from the database
	lastBlock, err := pdb.GetLastBlock()
	if err != nil {
		return fmt.Errorf("error retrieving last block: %v", err)
	}

	// initialize the block key to the previous block of the last block
	blockKey := lastBlock.PrevBlock

	// Loop through the blockchain to verify the integrity
	for blockKey != nil {
		currentBlock, err := pdb.GetBlock(blockKey)
		if err != nil {
			return fmt.Errorf("error retrieving block: %v", err)
		}

		if currentBlock.IsGenesisBlock() {
			if err := pdb.blockValidator.Validate(currentBlock, block.Block{}); err != nil {
				return fmt.Errorf("block validation failed: %v", err)
			}
			break
		}

		prevBlock, err := pdb.GetBlock(currentBlock.PrevBlock)
		if err != nil {
			return fmt.Errorf("previous block not found: %v", err)
		}

		if err := consensus.ValidateBlock(currentBlock, prevBlock); err != nil {
			return fmt.Errorf("block validation failed: %v", err)
		}

		blockKey = currentBlock.PrevBlock
	}

	pdb.log.Infoln("Blockchain integrity verified")
	return nil
}

// GetLastBlock retrieves the most recently added block from the database.
func (pdb *PebbleDB) GetLastBlock() (block.Block, error) {
	lastBlock, err := pdb.GetBlock([]byte("lastBlockKey"))
	if err != nil {
		return block.Block{}, fmt.Errorf("error retrieving last block: %v", err)
	}
	pdb.log.Debugln("last block retrieved successfully")
	return lastBlock, nil
}

// Close closes the database connection.
func (pdb *PebbleDB) Close() error {
	pdb.log.Infoln("closing pebble database")
	return pdb.db.Close()
}

// saveBlock serializes and stores a given block in the database.
func (pdb *PebbleDB) saveBlock(key []byte, b block.Block) error {
	blockBytes, err := b.Serialize()
	if err != nil {
		return fmt.Errorf("error serializing block: %v", err)
	}

	// Size of the serialized block
	serializedBlockSize := float64(len(blockBytes)) / 1024

	pdb.log.Debugf("Block size: %.2f KB", serializedBlockSize)
	err = pdb.db.Set(key, blockBytes, nil)
	if err != nil {
		return fmt.Errorf("error saving block to the database : %v", err)
	}

	key = []byte("lastBlockKey")

	// Delete the previous reference to the last block
	if err := pdb.db.Delete(key, nil); err != nil {
		return fmt.Errorf("error deleting last block reference: %v", err)
	}

	if err := pdb.db.Set(key, blockBytes, nil); err != nil {
		return fmt.Errorf("error updating last block reference: %v", err)
	}

	pdb.log.Infoln("block saved successfully")
	return nil
}

// addGenesisBlock adds the genesis block to the blockchain database.
func (pdb *PebbleDB) addGenesisBlock(b block.Block) error {
	// Compute the hash of the block to use as a key in the db
	key := block.ComputeHash(b)

	err := pdb.saveBlock(key, b)
	if err != nil {
		return fmt.Errorf("failed to add genesis block to the db: %s", err)
	}

	pdb.log.Debugln("Genesis block successfully added to the blockchain")
	return nil
}

// addNormalBlock adds a normal block to the blockchain database.
func (pdb *PebbleDB) addNormalBlock(b block.Block) error {
	key := block.ComputeHash(b)

	// Check if the block already exists in the db
	_, err := pdb.GetBlock(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			pdb.log.Warnf("Block not found in the db: %s\n", err)

			if err := pdb.saveBlock(key, b); err != nil {
				return fmt.Errorf("failed to add block to the db: %s", err)
			}

			pdb.log.Debugln("Block successfully added to the blockchain")
			return nil
		}

		return fmt.Errorf("failed to retrieve block from the db: %s", err)

	}

	return fmt.Errorf("block already exists in the blockchain")
}
