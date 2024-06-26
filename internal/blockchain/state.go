package blockchain

import "github.com/pierreleocadie/SecuraChain/internal/core/block"

type State interface {
	HandleBlock(block block.Block)
	SyncBlockchain()
	PostSync()
	GetCurrentStateName() string
}
