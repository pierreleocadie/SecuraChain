package blockmanager

import (
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/registry"
)

// AddBlockTransactionToRegistry adds the transactions of the given block to the indexing registry.
func AddBlockTransactionToRegistry(log *ipfsLog.ZapEventLogger, config *config.Config, b *block.Block) bool {
	for _, tx := range b.Transactions {
		switch tx.(type) {
		case *transaction.AddFileTransaction:
			addFileTransac := tx.(*transaction.AddFileTransaction)
			if err := registry.AddFileToRegistry(log, config, addFileTransac); err != nil {
				log.Errorln("Error adding file to registry: ", err)
				return false
			}

			log.Debugln("[AddBlockTransactionToRegistry] - File added to the registry")

		case *transaction.DeleteFileTransaction:
			deleteFileTransac := tx.(*transaction.DeleteFileTransaction)
			if err := registry.DeleteFileFromRegistry(log, config, deleteFileTransac); err != nil {
				log.Errorln("Error deleting file from registry: ", err)
				return false
			}

			log.Debugln("[AddBlockTransactionToRegistry] - File deleted from the registry")
		}
	}

	log.Debugln("Transactions of block updated to the registry")
	return true
}
