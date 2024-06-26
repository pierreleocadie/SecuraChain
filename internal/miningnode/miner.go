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
	ctx                         context.Context
	cancel                      context.CancelFunc
	blockchain                  *blockchain.Blockchain
	stopMiningChanNotCleaned    chan consensus.StopMiningSignal
	stopMiningChanCleaned       chan consensus.StopMiningSignal
	psh                         *node.PubSubHub
	transactionValidatorFactory consensus.TransactionValidatorFactory
	trxPool                     []transaction.Transaction
	mu                          sync.Mutex
	currentBlock                *block.Block
	previousBlock               *block.Block
	log                         *ipfsLog.ZapEventLogger
	ecdsaKeyPair                ecdsa.KeyPair
}

func NewMiner(log *ipfsLog.ZapEventLogger, psh *node.PubSubHub, transactionValidatorFactory consensus.TransactionValidatorFactory,
	blockchain *blockchain.Blockchain, stopMiningChanNotCleaned chan consensus.StopMiningSignal, ecdsaKeyPair ecdsa.KeyPair) *Miner {
	ctx, cancel := context.WithCancel(context.Background())
	miner := &Miner{
		ctx:                         ctx,
		cancel:                      cancel,
		blockchain:                  blockchain,
		stopMiningChanNotCleaned:    stopMiningChanNotCleaned,
		stopMiningChanCleaned:       make(chan consensus.StopMiningSignal),
		psh:                         psh,
		transactionValidatorFactory: transactionValidatorFactory,
		trxPool:                     make([]transaction.Transaction, 0),
		previousBlock:               &block.Block{},
		log:                         log,
		ecdsaKeyPair:                ecdsaKeyPair,
	}
	miner.currentBlock = block.NewBlock(miner.trxPool, nil, 1, miner.ecdsaKeyPair)
	blockchain.RegisterObserver(miner)
	return miner
}

func (m *Miner) Update(state string) {
	if state != "UpToDateState" {
		m.log.Debugln("Signal sent to stop mining because the blockchain is not up to date")
		m.stopMiningChanCleaned <- consensus.StopMiningSignal{Stop: true, Info: "Blockchain not up to date", BlockReceived: block.Block{}}
		m.cancel()
	} else {
		m.ctx, m.cancel = context.WithCancel(context.Background())
		go m.StartMining()
		go m.stopMiningChanCleaner()
	}
}

func (m *Miner) stopMiningChanCleaner() {
	m.log.Debug("Starting the stop mining signal chan cleaner")
	for {
		select {
		case <-m.ctx.Done():
			return
		case signal := <-m.stopMiningChanNotCleaned:
			if signal.Stop {
				if !reflect.DeepEqual(signal.BlockReceived, block.Block{}) {
					if signal.BlockReceived.Height >= m.currentBlock.Height {
						m.log.Debug("Signal cleaner - Block received have a height greater or equal to the current block")
						m.stopMiningChanCleaned <- signal
					}
				}
			}
		}
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
				err := copier.Copy(m.previousBlock, lastBlockStored)
				if err != nil {
					m.log.Errorln("Error copying the last block stored : ", err)
				}
				previousBlockHash := block.ComputeHash(*m.previousBlock)

				m.log.Infof("Last block stored on chain : %v at height %d", previousBlockHash, m.previousBlock.Height)
				m.mu.Lock()
				m.currentBlock = block.NewBlock(m.trxPool, previousBlockHash, m.previousBlock.Height+1, m.ecdsaKeyPair)
				m.trxPool = []transaction.Transaction{}
				m.mu.Unlock()
			} else {
				m.log.Infof("No block stored on chain")
			}
			m.log.Debug("Mining a new block")
			// m.currentBlock.MerkleRoot = m.currentBlock.ComputeMerkleRoot()
			stoppedEarly, blockReceivedEarly := consensus.MineBlock(m.currentBlock, m.stopMiningChanCleaned)
			if stoppedEarly {
				m.log.Infof("Signal received to stop mining - Signal : %v - %v", stoppedEarly, blockReceivedEarly)
				if reflect.DeepEqual(blockReceivedEarly, block.Block{}) {
					m.log.Info("Mining stopped early because synchronization is required")
					m.log.Info("Waiting for the blockchain to be synchronized with the network")
					m.log.Info("Putting the transactions back in the transaction pool")
					m.mu.Lock()
					m.trxPool = append(m.trxPool, m.currentBlock.Transactions...)
					m.mu.Unlock()
					continue
				}
				m.log.Info("Mining stopped early because of a new block received")
				lastBlockStored = block.Block{} // Reset the last block stored and be sure its not nil to avoid errors with copier
				err := copier.Copy(&lastBlockStored, blockReceivedEarly)
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
							m.mu.Lock()
							m.trxPool = append(m.trxPool, trxData)
							m.mu.Unlock()
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
			if !reflect.DeepEqual(m.previousBlock, &block.Block{}) {
				previousBlockHashEncoded := fmt.Sprintf("%x", block.ComputeHash(*m.previousBlock))
				m.log.Infoln("Previous block hash : ", previousBlockHashEncoded, " TIMESTAMP : ", m.previousBlock.Timestamp)
			}

			if err := m.blockchain.BlockValidator.Validate(*m.currentBlock, *m.previousBlock); err != nil {
				m.log.Warn("Block is invalid")
				// Return transactions of the current block to the transaction pool
				m.mu.Lock()
				m.trxPool = append(m.trxPool, m.currentBlock.Transactions...)
				m.mu.Unlock()
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
				if err != nil {
					m.log.Errorln("Error getting the last block stored : ", err)
				}
				if !reflect.DeepEqual(lastBlockStored, block.Block{}) && lastBlockStored.Height >= m.currentBlock.Height {
					break
				}
				time.Sleep(5 * time.Second)
			}
			// In order to avoid the case where we receive the signal for our own block and we stop mining
			// Clean the stopMiningChanCleaned
		clearer:
			for {
				select {
				case <-m.stopMiningChanCleaned:
				default:
					m.log.Debug("Channel stopMiningChanCleaned cleaned")
					break clearer // Break the clearer loop
				}
			}
		}
	}
}

func (m Miner) GetCurrentBlock() block.Block {
	return *m.currentBlock
}

func (m *Miner) HandleTransaction(trx transaction.Transaction) {
	trxValidator, err := m.transactionValidatorFactory.GetValidator(trx)
	if err != nil {
		m.log.Warn("Transaction type is not supported")
		return
	}
	if err := trxValidator.Validate(trx); err != nil {
		m.log.Warn("Transaction is invalid")
		return
	}

	m.mu.Lock()
	m.trxPool = append(m.trxPool, trx)
	m.log.Debugf("Transaction pool size : %d", len(m.trxPool))
	m.mu.Unlock()
}
