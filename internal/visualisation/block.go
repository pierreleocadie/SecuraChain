package visualisation

import (
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

type Block struct {
	Hash []byte `json:"hash"`
	block.Block
}

func NewBlock(b block.Block) Block {
	return Block{
		Hash:  block.ComputeHash(&b),
		Block: b,
	}
}
