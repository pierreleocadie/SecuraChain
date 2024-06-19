package registry

import (
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
)

// FileRegistry represents a registry of owners' files.
type FileData struct {
	Filename             []byte          `json:"filename"`
	Extension            []byte          `json:"extension"`
	FileSize             uint64          `json:"fileSize"`
	Checksum             []byte          `json:"checksum"`
	Providers            []peer.AddrInfo `json:"providers"`
	FileCid              cid.Cid         `json:"fileCid"`
	TransactionTimestamp int64           `json:"transactionTimestamp"`
}

func NewFileData(addFileTransac transaction.AddFileTransaction) FileData {
	return FileData{
		Filename:             addFileTransac.Filename,
		Extension:            addFileTransac.Extension,
		FileSize:             addFileTransac.FileSize,
		Checksum:             addFileTransac.Checksum,
		Providers:            []peer.AddrInfo{addFileTransac.IPFSStorageNodeAddrInfo},
		FileCid:              addFileTransac.FileCid,
		TransactionTimestamp: addFileTransac.AnnouncementTimestamp,
	}
}
