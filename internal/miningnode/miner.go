package miningnode

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/jinzhu/copier"
	"github.com/pierreleocadie/SecuraChain/internal/blockchain"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

type Miner struct {
	ctx            context.Context
	cancel         context.CancelFunc
	blockchain     *blockchain.Blockchain
	mu             sync.Mutex
	stopMiningChan chan consensus.StopMiningSignal
	psh            *node.PubSubHub
	trxPool        []transaction.Transaction
	currentBlock   *block.Block
	previousBlock  *block.Block
	log            *ipfsLog.ZapEventLogger
	ecdsaKeyPair   ecdsa.KeyPair
}

func NewMiner(log *ipfsLog.ZapEventLogger, psh *node.PubSubHub, blockchain *blockchain.Blockchain,
	stopMiningChan chan consensus.StopMiningSignal, ecdsaKeyPair ecdsa.KeyPair) *Miner {
	ctx, cancel := context.WithCancel(context.Background())
	miner := &Miner{
		ctx:            ctx,
		cancel:         cancel,
		blockchain:     blockchain,
		stopMiningChan: stopMiningChan,
		psh:            psh,
		trxPool:        make([]transaction.Transaction, 0),
		log:            log,
		ecdsaKeyPair:   ecdsaKeyPair,
	}
	blockchain.RegisterObserver(miner)
	return miner
}

func (m *Miner) Update(state string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state != "UpToDateState" {
		m.stopMiningChan <- consensus.StopMiningSignal{Stop: true, BlockReceived: block.Block{}}
		m.cancel()
	} else {
		m.ctx, m.cancel = context.WithCancel(context.Background())
		go m.StartMining()
	}
}

func (m *Miner) StartMining() {
	m.log.Infoln("Mining started")
	lastBlockStored, err := m.blockchain.Database.GetLastBlock()
	if err != nil {
		m.log.Errorln("Error getting the last block stored : ", err)
	}
	for {
		select {
		case <-m.ctx.Done():
			m.log.Infoln("Mining stopped")
			return
		default:
			if !reflect.DeepEqual(lastBlockStored, block.Block{}) {
				lastBlockStored, err = m.blockchain.Database.GetLastBlock()
				if err != nil {
					m.log.Errorln("Error getting the last block stored : ", err)
				}
				m.previousBlock = &block.Block{}
				err := copier.Copy(m.previousBlock, &lastBlockStored)
				if err != nil {
					m.log.Errorln("Error copying the last block stored : ", err)
				}
				previousBlockHash := block.ComputeHash(*m.previousBlock)

				m.log.Infof("Last block stored on chain : %v at height %d", previousBlockHash, m.previousBlock.Height)
				m.currentBlock = block.NewBlock(m.trxPool, previousBlockHash, m.previousBlock.Height+1, m.ecdsaKeyPair)
				m.trxPool = []transaction.Transaction{}
			} else {
				m.log.Infof("No block stored on chain")
			}
			m.log.Debug("Mining a new block")
			// m.currentBlock.MerkleRoot = m.currentBlock.ComputeMerkleRoot()
			stoppedEarly, blockReceivedEarly := consensus.MineBlock(m.currentBlock, m.stopMiningChan)
			if stoppedEarly {
				if reflect.DeepEqual(blockReceivedEarly, block.Block{}) {
					m.log.Info("Mining stopped early because synchronization is required")
					m.log.Info("Waiting for the blockchain to be synchronized with the network")
					m.log.Info("Putting the transactions back in the transaction pool")
					m.trxPool = append(m.trxPool, m.currentBlock.Transactions...)
					continue
				}
				m.log.Info("Mining stopped early because of a new block received")
				lastBlockStored = block.Block{} // Reset the last block stored and be sure its not nil to avoid errors with copier
				err := copier.Copy(lastBlockStored, blockReceivedEarly)
				if err != nil {
					m.log.Errorln("Error copying the block received early : ", err)
				}

				// If there are transactions that are in my current block but not in the received block
				// I need to put them back in the transaction pool to be sure they are mined
				// We can simply compare the merkle root of the two blocks to know if there are transactions that are not in the received block
				if !bytes.Equal(m.currentBlock.MerkleRoot, blockReceivedEarly.MerkleRoot) {
					m.log.Debug("Some transactions in the current block are not in the received block")
					// We can simply use TransactionID to know which transactions are not in the received block
					// We can then put them back in the transaction pool
					currentBlockTransactionIDs := m.currentBlock.GetTransactionIDsMap()
					blockReceivedEarlyTransactionIDs := blockReceivedEarly.GetTransactionIDsMap()
					for trxID, trxData := range currentBlockTransactionIDs {
						if _, ok := blockReceivedEarlyTransactionIDs[trxID]; !ok {
							m.trxPool = append(m.trxPool, trxData)
						}
					}
				}
				continue
			}

			err = m.currentBlock.SignBlock(m.ecdsaKeyPair)
			if err != nil {
				m.log.Errorln("Error signing the block : ", err)
				continue
			}

			currentBlockHashEncoded := fmt.Sprintf("%x", block.ComputeHash(*m.currentBlock))
			m.log.Infoln("Current block hash : ", currentBlockHashEncoded, " TIMESTAMP : ", m.currentBlock.Timestamp)
			if m.previousBlock != nil {
				previousBlockHashEncoded := fmt.Sprintf("%x", block.ComputeHash(*m.previousBlock))
				m.log.Infoln("Previous block hash : ", previousBlockHashEncoded, " TIMESTAMP : ", m.previousBlock.Timestamp)
			}

			if err := m.blockchain.BlockValidator.Validate(*m.currentBlock, *m.previousBlock); err != nil {
				m.log.Warn("Block is invalid")
				// Return transactions of the current block to the transaction pool
				m.trxPool = append(m.trxPool, m.currentBlock.Transactions...)
				continue
			}

			serializedBlock, err := m.currentBlock.Serialize()
			if err != nil {
				m.log.Errorln("Error serializing the block : ", err)
				continue
			}
			if err = m.psh.BlockAnnouncementTopic.Publish(m.ctx, serializedBlock); err != nil {
				m.log.Errorln("Error publishing the block : ", err)
				continue
			}

			m.log.Infof("Block mined and published with hash %v at height %d", block.ComputeHash(*m.currentBlock), m.currentBlock.Height)

			// To be sure we retrieve the last block stored
			for {
				lastBlockStored, err = m.blockchain.Database.GetLastBlock()
				if !reflect.DeepEqual(lastBlockStored, block.Block{}) && lastBlockStored.Height >= m.currentBlock.Height {
					break
				}
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (m Miner) GetCurrentBlock() block.Block {
	return *m.currentBlock
}
