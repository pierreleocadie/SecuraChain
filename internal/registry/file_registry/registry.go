package fileregistry

import "github.com/pierreleocadie/SecuraChain/internal/core/transaction"

type FileRegistry interface {
	Add(addFileTransac transaction.AddFileTransaction) error
	Get(myPublicKey string) []FileData
	Delete(deleteFileTransac transaction.DeleteFileTransaction) error
}
