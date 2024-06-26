package client

import (
	"github.com/ipfs/go-cid"
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

func DeleteFile(log *ipfsLog.ZapEventLogger, fileCid cid.Cid, ecdsaKeyPair ecdsa.KeyPair, deleteFileTrxChan chan transaction.Transaction) {
	deleteFileTrx := transaction.NewDeleteFileTransaction(ecdsaKeyPair, fileCid)
	deleteFileTrxChan <- deleteFileTrx
}
