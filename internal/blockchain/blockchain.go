package blockchain

import (
	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	blockregistry "github.com/pierreleocadie/SecuraChain/internal/registry/block_registry"
	fileregistry "github.com/pierreleocadie/SecuraChain/internal/registry/file_registry"
)

type Blockchain struct {
	UpToDateState  *UpToDateState
	SyncingState   *SyncingState
	PostSyncState  *PostSyncState
	currentState   State
	pendingBlocks  []block.Block
	nodeBlacklist  []string
	ipfsNode       *ipfs.IPFSNode
	blockValidator consensus.BlockValidator
	database       blockchaindb.BlockchainDB
	fileRegistry   fileregistry.FileRegistry
	blockRegistry  blockregistry.BlockRegistry
	log            *ipfsLog.ZapEventLogger
	config         *config.Config
}

func NewBlockchain(log *ipfsLog.ZapEventLogger, config *config.Config, ipfsNode *ipfs.IPFSNode,
	blockValidator consensus.BlockValidator, database blockchaindb.BlockchainDB,
	blockRegistry blockregistry.BlockRegistry, fileRegistry fileregistry.FileRegistry) *Blockchain {
	blockchain := &Blockchain{
		pendingBlocks:  make([]block.Block, 0),
		database:       database,
		log:            log,
		config:         config,
		ipfsNode:       ipfsNode,
		blockValidator: blockValidator,
		fileRegistry:   fileRegistry,
		blockRegistry:  blockRegistry,
	}
	blockchain.UpToDateState = &UpToDateState{blockchain: blockchain}
	blockchain.SyncingState = &SyncingState{blockchain: blockchain}
	blockchain.PostSyncState = &PostSyncState{blockchain: blockchain}
	blockchain.currentState = blockchain.UpToDateState
	return blockchain
}

func (n *Blockchain) SetState(state State) {
	n.currentState = state
}

func (n *Blockchain) HandleBlock(block block.Block) {
	n.currentState.HandleBlock(block)
}

func (n *Blockchain) SyncBlockchain() {
	n.currentState.SyncBlockchain()
}

func (n *Blockchain) PostSync() {
	n.currentState.PostSync()
}
