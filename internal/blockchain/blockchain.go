package blockchain

import (
	"context"
	"sync"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/internal/observer"
	blockregistry "github.com/pierreleocadie/SecuraChain/internal/registry/block_registry"
	fileregistry "github.com/pierreleocadie/SecuraChain/internal/registry/file_registry"
)

type Blockchain struct {
	Ctx            context.Context
	UpToDateState  *UpToDateState
	SyncingState   *SyncingState
	PostSyncState  *PostSyncState
	currentState   State
	pendingBlocks  []block.Block
	nodeBlacklist  []string
	ipfsNode       *ipfs.IPFSNode
	pubsubHub      *node.PubSubHub
	BlockValidator consensus.BlockValidator
	Database       blockchaindb.BlockchainDB
	fileRegistry   fileregistry.FileRegistry
	blockRegistry  blockregistry.BlockRegistry
	log            *ipfsLog.ZapEventLogger
	config         *config.Config
	StopMiningChan chan consensus.StopMiningSignal

	observers []observer.Observer
	mu        sync.Mutex
}

func NewBlockchain(log *ipfsLog.ZapEventLogger, config *config.Config, ctx context.Context, ipfsNode *ipfs.IPFSNode,
	pubsubHub *node.PubSubHub, blockValidator consensus.BlockValidator, database blockchaindb.BlockchainDB,
	blockRegistry blockregistry.BlockRegistry, fileRegistry fileregistry.FileRegistry, stopMiningChan chan consensus.StopMiningSignal) *Blockchain {
	blockchain := &Blockchain{
		Ctx:            ctx,
		pendingBlocks:  make([]block.Block, 0),
		Database:       database,
		log:            log,
		config:         config,
		ipfsNode:       ipfsNode,
		BlockValidator: blockValidator,
		fileRegistry:   fileRegistry,
		blockRegistry:  blockRegistry,
		pubsubHub:      pubsubHub,
		StopMiningChan: stopMiningChan,
		observers:      make([]observer.Observer, 0),
	}
	blockchain.UpToDateState = &UpToDateState{name: "UpToDateState", blockchain: blockchain}
	blockchain.SyncingState = &SyncingState{name: "SyncingState", blockchain: blockchain}
	blockchain.PostSyncState = &PostSyncState{name: "PostSyncState", blockchain: blockchain}
	blockchain.currentState = blockchain.UpToDateState
	return blockchain
}

func (n *Blockchain) SetState(state State) {
	n.currentState = state
	n.NotifyObservers()
}

func (n *Blockchain) GetState() State {
	return n.currentState
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

func (n *Blockchain) GetCurrentStateName() string {
	return n.currentState.GetCurrentStateName()
}

func (n *Blockchain) NotifyObservers() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, observer := range n.observers {
		observer.Update(n.currentState.GetCurrentStateName())
	}
}

func (n *Blockchain) RegisterObserver(observer observer.Observer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.observers = append(n.observers, observer)
}

func (n *Blockchain) RemoveObserver(observer observer.Observer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i, obs := range n.observers {
		if obs == observer {
			n.observers = append(n.observers[:i], n.observers[i+1:]...)
			break
		}
	}
}
