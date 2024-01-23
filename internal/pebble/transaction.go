package pebble

import (
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// TransactionDatabase est une interface pour interagir avec la base de données des transactions.
type TransactionDatabase interface {
	SaveTransaction(key []byte, bx block.Block) error
	GetTransaction(key []byte) (block.Block, error)
	Close() error
}

// PebbleTransactionDB est une implémentation de l'interface TransactionDatabase utilisant la bibliothèque Pebble.
type PebbleTransactionDB struct {
	db *pebble.DB
}

// NewPebbleTransactionDB crée une nouvelle instance de PebbleTransactionDB.
func NewPebbleTransactionDB(dbPath string) (*PebbleTransactionDB, error) {
	options := &pebble.Options{}
	db, err := pebble.Open(dbPath, options)
	if err != nil {
		return nil, err
	}
	return &PebbleTransactionDB{db: db}, nil
}

// SaveTransaction enregistre une transaction dans la base de données.
func (pdb *PebbleTransactionDB) SaveBlock(key []byte, bx *block.Block) error {
	bxBytes, err := json.Marshal(bx)
	if err != nil {
		return fmt.Errorf("erreur de sérialisation de la transaction: %v", err)
	}

	err = pdb.db.Set(key, bxBytes, pebble.Sync)
	if err != nil {
		return fmt.Errorf("erreur d'enregistrement de la transaction: %v", err)
	}
	return nil
}

// GetTransaction récupère une transaction à partir de la base de données.
func (pdb *PebbleTransactionDB) GetBlock(key []byte) (*block.Block, error) {
	bxBytes, closer, err := pdb.db.Get(key)
	if err != nil {
		return nil, fmt.Errorf("erreur de récupération de la transaction: %v", err)
	}
	defer closer.Close()

	blockDeserialize, err := block.DeserializeBlock(bxBytes)
	if err != nil {
		return nil, fmt.Errorf("erreur de désérialisation de la transaction: %v", err)
	}

	return blockDeserialize, nil
}

// Close ferme la connexion à la base de données.
func (pdb *PebbleTransactionDB) Close() error {
	return pdb.db.Close()
}
