package blockchaindb

import (
	"fmt"

	"github.com/cockroachdb/pebble"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
)

// BlockchainStorage defines an interface for interacting with the blockchain storage.
type BlockchainStorage interface {
	SaveBlock(log *ipfsLog.ZapEventLogger, key []byte, b *block.Block) error
	GetBlock(log *ipfsLog.ZapEventLogger, key []byte) (*block.Block, error)
	VerifyIntegrity(log *ipfsLog.ZapEventLogger) (bool, error)
	GetLastBlock(log *ipfsLog.ZapEventLogger) *block.Block
	Close(log *ipfsLog.ZapEventLogger) error
}

// BlockchainDB wraps a Pebble database instance to store blockchain data.
type BlockchainDB struct {
	db *pebble.DB
}

// BlockchainDB wraps a Pebble database instance to store blockchain data.
func NewBlockchainDB(log *ipfsLog.ZapEventLogger, dbPath string) (*BlockchainDB, error) {
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Errorln("failed to open pebble database")
		return nil, fmt.Errorf("failed to open pebble database : %v", err)
	}
	log.Debugln("Pebble database opened successfully")
	return &BlockchainDB{db: db}, nil
}

// SaveBlock serializes and stores a given block in the database.
func (pdb *BlockchainDB) SaveBlock(log *ipfsLog.ZapEventLogger, key []byte, b *block.Block) error {
	blockBytes, err := b.Serialize()
	if err != nil {
		log.Errorln("error serializing block")
		return fmt.Errorf("error serializing block: %v", err)
	}

	err = pdb.db.Set(key, blockBytes, pebble.Sync)
	if err != nil {
		log.Errorln("error saving block to the database")
		return fmt.Errorf("error saving block to the database : %v", err)
	}

	log.Infoln("block saved successfully")
	return pdb.saveLastBlock(log, b)
}

// GetBlock retrieves a block from the database using its key.
func (pdb *BlockchainDB) GetBlock(log *ipfsLog.ZapEventLogger, key []byte) (*block.Block, error) {
	blockData, closer, err := pdb.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			log.Warn("block not exists in the database")
			return nil, nil
		}
		log.Errorln("error retrieving block")
		return nil, fmt.Errorf("error retrieving block : %v", err)
	}
	defer closer.Close()

	b, err := block.DeserializeBlock(blockData)
	if err != nil {
		log.Errorln("error deserializing block")
		return nil, fmt.Errorf("error deserializing block: %v", err)
	}

	log.Debugln("block retrieved successfully")
	return b, nil
}

// VerifyIntegrity verifies the integrity of the blockchain stored in the BlockchainDB based on the last block.
func (pdb *BlockchainDB) VerifyIntegrity(log *ipfsLog.ZapEventLogger) bool {

	for blockKey := pdb.GetLastBlock(log).PrevBlock; blockKey != nil; {
		currentBlock, err := pdb.GetBlock(log, blockKey)
		if err != nil {
			log.Errorln("error retrieving block")
			return false
		}

		prevBlock, err := pdb.GetBlock(log, currentBlock.PrevBlock)
		if err != nil {
			log.Errorln("previous block not found")
			return false
		}

		if !consensus.ValidateBlock(currentBlock, prevBlock) {
			log.Errorln("block validation failed")
			return false
		}

		blockKey = currentBlock.PrevBlock
	}

	log.Infoln("Blockchain integrity verified")
	return true
}

// saveLastBlock updates the reference to the last block in the blockchain.
func (pdb *BlockchainDB) saveLastBlock(log *ipfsLog.ZapEventLogger, lastBlock *block.Block) error {
	// Delete the previous reference to the last block
	if err := pdb.db.Delete([]byte("lastBlockKey"), pebble.Sync); err != nil {
		log.Errorln("error deleting last block reference")
		return fmt.Errorf("error deleting last block reference: %v", err)
	}

	serializedBlock, err := lastBlock.Serialize()
	if err != nil {
		log.Errorln("error serializing block")
		return fmt.Errorf("error serializing block: %v", err)
	}

	if err := pdb.db.Set([]byte("lastBlockKey"), serializedBlock, pebble.Sync); err != nil {
		log.Errorln("error updating last block reference")
		return fmt.Errorf("error updating last block reference: %v", err)
	}

	log.Debugln("last block reference updated successfully")
	return nil
}

// GetLastBlock retrieves the most recently added block from the database.
func (pdb *BlockchainDB) GetLastBlock(log *ipfsLog.ZapEventLogger) *block.Block {
	lastBlock, err := pdb.GetBlock(log, []byte("lastBlockKey"))
	if err != nil {
		log.Errorln("error retrieving last block")
		return nil
	}
	log.Debugln("last block retrieved successfully")
	return lastBlock
}

// Close closes the database connection.
func (pdb *BlockchainDB) Close(log *ipfsLog.ZapEventLogger) error {
	log.Infoln("closing pebble database")
	return pdb.db.Close()
}
