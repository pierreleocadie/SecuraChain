package blockregistry

import (
	"github.com/ipfs/boxo/path"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

type BlockRegistry interface {
	Add(b block.Block, fileCid path.ImmutablePath, provider peer.AddrInfo) error
}
