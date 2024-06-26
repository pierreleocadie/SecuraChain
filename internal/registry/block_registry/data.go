package blockregistry

import (
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// BlockData represents the data associated with a block stored with IPFS.
type BlockData struct {
	ID       uint32        `json:"ID"`
	Key      []byte        `json:"key"`
	BlockCid cid.Cid       `json:"cid"`
	Provider peer.AddrInfo `json:"provider"`
}

func NewBlockData(b block.Block, fileCid path.ImmutablePath, provider peer.AddrInfo) BlockData {
	return BlockData{
		ID:       b.Header.Height,
		Key:      block.ComputeHash(b),
		BlockCid: fileCid.RootCid(),
		Provider: provider,
	}
}
