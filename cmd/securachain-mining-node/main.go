package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"
	"reflect"
	"time"

	"github.com/jinzhu/copier"
	"github.com/pierreleocadie/SecuraChain/internal/blockchain"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	blockregistry "github.com/pierreleocadie/SecuraChain/internal/registry/block_registry"
	fileregistry "github.com/pierreleocadie/SecuraChain/internal/registry/file_registry"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
)

var (
	yamlConfigFilePath          = flag.String("config", "", "Path to the yaml config file")
	generateKeys                = flag.Bool("genKeys", false, "Generate new ECDSA and AES keys to the paths specified in the config file")
	waitingTime                 = flag.Int("waitingTime", 60, "Time to wait before starting the mining process and the transaction processing")
	blockReceived               = make(chan block.Block, 100)
	requiresSync                = false
	requiresPostSync            = false
	blockProcessingEnabled      = true
	trxPool                     = []transaction.Transaction{}
	stopMiningChan              = make(chan consensus.StopMiningSignal)
	stopMiningChanCleaned       = make(chan consensus.StopMiningSignal)
	transactionValidatorFactory = consensus.DefaultTransactionValidatorFactory{}
	genesisValidator            = consensus.DefaultGenesisBlockValidator{}
)

func main() {
	log := ipfsLog.Logger("mining-node")
	if err := ipfsLog.SetLogLevel("mining-node", "DEBUG"); err != nil {
		log.Errorln("Failed to set log level : ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flag.Parse()

	// Load the config file
	cfg := node.LoadConfig(yamlConfigFilePath, log)

	// Generate and/or load the keys
	if *generateKeys {
		node.GenerateKeys(cfg, log)
	}
	ecdsaKeyPair, _ := node.LoadKeys(cfg, log)

	// Create the SecuraChain data directory
	securaChainDataDirectory, err := node.CreateSecuraChainDataDirectory(log, cfg)
	if err != nil {
		log.Panicf("Error creating the SecuraChain data directory : %s\n", err)
	}

	// Initialize the block validator
	blockValidator := consensus.NewDefaultBlockValidator(genesisValidator, transactionValidatorFactory)

	// For the mining process
	var previousBlock *block.Block = &block.Block{}
	currentBlock := block.NewBlock(trxPool, nil, 1, ecdsaKeyPair)

	/*
	* IPFS NODE
	 */
	IPFSNode := ipfs.NewIPFSNode(ctx, log, cfg)

	/*
	 * IPFS MEMORY MANAGEMENT
	 */
	storageMax, err := IPFSNode.ChangeStorageMax(cfg.MemorySpace)
	if err != nil {
		log.Warnf("Failed to change storage max: %s", err)
	}
	if storageMax {
		log.Infof("Storage max changed: %v", storageMax)
	}

	freeMemorySpace, err := IPFSNode.FreeMemoryAvailable()
	if err != nil {
		log.Warnf("Failed to get free memory available: %s", err)
	}
	log.Infof("Free memory available: %v", freeMemorySpace)

	memoryUsedGB, err := IPFSNode.MemoryUsed()
	if err != nil {
		log.Warnf("Failed to get memory used: %s", err)
	}
	log.Infof("Memory used: %v", memoryUsedGB)

	/*
	* NODE LIBP2P
	 */
	// Initialize the node
	host := node.Initialize(log, *cfg)
	defer host.Close()
	log.Debugf("Storage node initizalized with PeerID: %s", host.ID().String())

	/*
	* DHT DISCOVERY
	 */
	node.SetupDHTDiscovery(ctx, cfg, host, false, log)

	/*
	* PUBSUB TOPICS AND SUBSCRIPTIONS
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// KeepRelayConnectionAlive
	keepRelayConnectionAliveTopic, err := ps.Join(cfg.KeepRelayConnectionAliveStringFlag)
	if err != nil {
		log.Warnf("Failed to join KeepRelayConnectionAlive topic: %s", err)
	}

	// Subscribe to KeepRelayConnectionAlive topic
	subKeepRelayConnectionAlive, err := keepRelayConnectionAliveTopic.Subscribe()
	if err != nil {
		log.Warnf("Failed to subscribe to KeepRelayConnectionAlive topic: %s", err)
	}

	// NetworkVisualisation
	networkVisualisationTopic, err := ps.Join(cfg.NetworkVisualisationStringFlag)
	if err != nil {
		log.Warnf("Failed to join NetworkVisualisation topic: %s", err)
	}

	// Join the topic BlockAnnouncementStringFlag
	blockAnnouncementTopic, err := ps.Join(cfg.BlockAnnouncementStringFlag)
	if err != nil {
		log.Panicf("Failed to join block announcement topic : %s\n", err)
	}

	subBlockAnnouncement, err := blockAnnouncementTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to block announcement topic : %s\n", err)
	}

	// Join the topic to ask for the json file of the blockchain
	askingBlockchainTopic, err := ps.Join(cfg.AskingBlockchainStringFlag)
	if err != nil {
		log.Panicf("Failed to join AskingBlockchain topic : %s\n", err)
	}

	// Subscribe to the topic to ask for the json file of the blockchain
	subAskingBlockchain, err := askingBlockchainTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to AskingBlockchain topic : %s\n", err)
	}

	// Join the topic to receive the json file of the blockchain
	receiveBlockchainTopic, err := ps.Join(cfg.ReceiveBlockchainStringFlag)
	if err != nil {
		log.Panicf("Failed to join ReceiveBlockchain topic : %s\n", err)
	}

	// Subscribe to the topic to receive the json file of the blockchain
	subReceiveBlockchain, err := receiveBlockchainTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to ReceiveBlockchain topic : %s\n", err)
	}

	// Join the topic to ask for my files
	askMyFilesTopic, err := ps.Join(cfg.AskMyFilesStringFlag)
	if err != nil {
		log.Panicf("Failed to join AskMyFiles topic : %s\n", err)
	}

	// Subscribe to the topic to ask for my files
	subAskMyFiles, err := askMyFilesTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to AskMyFiles topic : %s\n", err)
	}

	// Join the topic to send the files of the owner
	sendFilesTopic, err := ps.Join(cfg.SendFilesStringFlag)
	if err != nil {
		log.Panicf("Failed to join SendFiles topic : %s\n", err)
	}

	// Join the topic StorageNodeResponse to receive transactions
	storageNodeResponseTopic, err := ps.Join(cfg.StorageNodeResponseStringFlag)
	if err != nil {
		log.Panicf("Failed to join StorageNodeResponse topic : %s\n", err)
	}

	// Subscribe to the topic StorageNodeResponse to receive transactions
	subStorageNodeResponse, err := storageNodeResponseTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to StorageNodeResponse topic : %s\n", err)
	}

	pubsubHub := &node.PubSubHub{
		ClientAnnouncementTopic:       nil,
		ClientAnnouncementSub:         nil,
		StorageNodeResponseTopic:      storageNodeResponseTopic,
		StorageNodeResponseSub:        subStorageNodeResponse,
		KeepRelayConnectionAliveTopic: keepRelayConnectionAliveTopic,
		KeepRelayConnectionAliveSub:   subKeepRelayConnectionAlive,
		BlockAnnouncementTopic:        blockAnnouncementTopic,
		BlockAnnouncementSub:          subBlockAnnouncement,
		AskingBlockchainTopic:         askingBlockchainTopic,
		AskingBlockchainSub:           subAskingBlockchain,
		ReceiveBlockchainTopic:        receiveBlockchainTopic,
		ReceiveBlockchainSub:          subReceiveBlockchain,
		AskMyFilesTopic:               askMyFilesTopic,
		AskMyFilesSub:                 subAskMyFiles,
		SendMyFilesTopic:              sendFilesTopic,
		SendMyFilesSub:                nil,
		NetworkVisualisationTopic:     networkVisualisationTopic,
		NetworkVisualisationSub:       nil,
	}

	/*
	* BLOCK REGISTRY
	 */
	blockRegistryPath := filepath.Join(securaChainDataDirectory, cfg.BlockRegistryPath)
	blockRegistryManager := blockregistry.NewDefaultBlockRegistryManager(log, cfg, blockRegistryPath)
	blockRegistry, err := blockregistry.NewDefaultBlockRegistry(log, cfg, blockRegistryManager)
	if err != nil {
		log.Warnln("Error creating or opening a block registry : %s\n", err)
	}

	/*
	 * FILE REGISTRY
	 */
	fileRegistryPath := filepath.Join(securaChainDataDirectory, cfg.FileRegistryPath)
	fileRegistryManager := fileregistry.NewDefaultFileRegistryManager(log, cfg, fileRegistryPath)
	fileRegistry, err := fileregistry.NewDefaultFileRegistry(log, cfg, fileRegistryManager)
	if err != nil {
		log.Warnln("Error creating or opening a file registry : %s\n", err)
	}

	/*
	 * BLOCKCHAIN DATABASE
	 */
	// Create the blockchain directory
	blockchainDBPath := filepath.Join(securaChainDataDirectory, "blockchain")
	blockchainDB, err := blockchaindb.NewPebbleDB(log, blockchainDBPath, blockValidator)
	if err != nil {
		log.Debugln("Error creating or opening a database : %s\n", err)
	}

	/*
	 * BLOCKCHAIN
	 */
	chain := blockchain.NewBlockchain(log, cfg, ctx, IPFSNode, pubsubHub, blockValidator, blockchainDB, blockRegistry, fileRegistry, stopMiningChan)

	/*
	 * SERVICES
	 */

	node.PubsubKeepRelayConnectionAlive(ctx, pubsubHub, host, cfg, log)

	// Service 1 : Receiption of blocks
	go func() {
		for {
			msg, err := subBlockAnnouncement.Next(ctx)
			if err != nil {
				log.Errorln("error getting block announcement message: ", err)
				continue
			}
			log.Debugln("Received block announcement message from ", msg.GetFrom().String())
			log.Debugln("Received block : ", string(msg.Data), " - State : ", chain.GetState().GetCurrentStateName())
			blockAnnounced, err := block.DeserializeBlock(msg.Data)
			if err != nil {
				log.Errorln("Error deserializing the block : ", err)
				continue
			}
			log.Debugln("Block received - 1: ", blockAnnounced)
			blockReceived <- blockAnnounced
			log.Debugln("Number of blocs in the channel : ", len(blockReceived))
		}
	}()

	// Service 2: Handle the block received
	go func() {
		for {
			block := <-blockReceived
			log.Debugln("Block received - 2: ", block)
			chain.HandleBlock(block)
		}
	}()

	// Service 3 : synchronization
	go func() {
		for {
			chain.SyncBlockchain()
		}
	}()

	// Service 4 : Post-synchronization
	go func() {
		for {
			chain.PostSync()
		}
	}()

	// Service 5 : Sending registry of the blockchain
	go func() {
		for {
			msg, err := subAskingBlockchain.Next(ctx)
			if err != nil {
				log.Debugln("Error getting message from the network : ", err)
				break
			}

			if msg.GetFrom().String() == host.ID().String() {
				continue
			}
			log.Debugln("Blockchain asked by a peer ", msg.GetFrom().String())

			// Send the registry of the blockchain
			blockRegistryBytes, err := json.Marshal(blockRegistry)
			if err != nil {
				log.Errorln("Error serializing the registry of the blockchain : ", err)
				continue
			}
			if err := receiveBlockchainTopic.Publish(ctx, blockRegistryBytes); err != nil {
				log.Errorln("Error publishing the registry of the blockchain : ", err)
				continue
			}
		}
	}()

	// Service 6 : Sending the files of the address given
	go func() {
		for {
			msg, err := subAskMyFiles.Next(ctx)
			if err != nil {
				log.Debugln("Error getting message from the network : ", err)
				break
			}

			if msg.GetFrom().String() == host.ID().String() {
				continue
			}
			log.Debugln("Files asked by a peer ", msg.GetFrom().String())

			// Send the files of the owner
			ownerAddressStr := fmt.Sprintf("%x", msg.Data)
			ownerFiles := fileRegistry.Get(ownerAddressStr)
			messageToSend := fileregistry.Message{OwnerPublicKey: ownerAddressStr, Registry: ownerFiles}
			ownerFilesBytes, err := json.Marshal(messageToSend)
			if err != nil {
				log.Errorln("Error serializing my files : ", err)
				continue
			}
			if err := sendFilesTopic.Publish(ctx, ownerFilesBytes); err != nil {
				log.Errorln("Error publishing indexing registry : ", err)
				continue
			}
		}
	}()

	// Handle the transactions received from the storage nodes
	go func() {
		log.Debugf("Waiting for %d seconds before starting the transaction processing", *waitingTime)
		time.Sleep(time.Duration(*waitingTime) * time.Second)
		for {
			msg, err := subStorageNodeResponse.Next(ctx)
			if err != nil {
				log.Debugln("Error getting message from the network : ", err)
				break
			}

			trx, err := transaction.DeserializeTransaction(msg.Data)
			if err != nil {
				log.Debugln("Error deserializing the transaction : ", err)
				continue
			}

			trxValidator, err := transactionValidatorFactory.GetValidator(trx)
			if err != nil {
				log.Errorln("Error getting the transaction validator : ", err)
				continue
			}
			if err := trxValidator.Validate(trx); err != nil {
				log.Warnln("Transaction is invalid : ", err)
				continue
			}

			trxPool = append(trxPool, trx)
			log.Debugf("Transaction pool size : %d", len(trxPool))
		}
	}()

	go func() {
		log.Debug("Starting the stop mining signal chan cleaner")
		for {
			select {
			case <-ctx.Done():
				return
			case signal := <-stopMiningChan:
				if signal.Stop {
					if !reflect.DeepEqual(signal.BlockReceived, block.Block{}) {
						if signal.BlockReceived.Height >= currentBlock.Height {
							log.Debug("Signal cleaner - Block received have a height greater or equal to the current block")
							stopMiningChanCleaned <- signal
						}
					} else {
						log.Debug("Signal cleaner - Block received is nil - signal for synchronization")
						stopMiningChanCleaned <- signal
					}
				}
			}
		}
	}()

	// Mining block
	go func() {
		log.Debugf("Waiting for %d seconds before starting the transaction processing", *waitingTime)
		time.Sleep(time.Duration(*waitingTime) * time.Second)
		log.Info("Mining process started")
		lastBlockStored, err := blockchainDB.GetLastBlock()
		if err != nil {
			log.Errorln("Error getting the last block stored : ", err)
		}
		for {
		chan1:
			for {
				select {
				case <-stopMiningChan:
				default:
					log.Debug("Channel stopMiningChanNotCleaned cleaned before starting the cleaner")
					break chan1 // Break the clearer loop
				}
			}
		chan2:
			for {
				select {
				case <-stopMiningChanCleaned:
				default:
					log.Debug("Channel stopMiningChanCleaned cleaned before starting mining")
					break chan2 // Break the clearer loop
				}
			}
			// The mining process requires the blockchain to be up to date
			if !blockProcessingEnabled || requiresSync || requiresPostSync {
				continue
			}
			if !reflect.DeepEqual(lastBlockStored, block.Block{}) {
				lastBlockStored, err = blockchainDB.GetLastBlock()
				if err != nil {
					log.Errorln("Error getting the last block stored : ", err)
				}

				previousBlock = &block.Block{}
				// Copy(desination, source) -> destination have to be a pointer
				err := copier.Copy(previousBlock, lastBlockStored)
				if err != nil {
					log.Errorln("Error copying the last block stored : ", err)
				}
				previousBlockHash := block.ComputeHash(*previousBlock)

				log.Infof("Last block stored on chain : %v at height %d", previousBlockHash, previousBlock.Height)
				currentBlock = block.NewBlock(trxPool, previousBlockHash, previousBlock.Height+1, ecdsaKeyPair)
				trxPool = []transaction.Transaction{}
			} else {
				log.Infof("No block stored on chain")
			}

			log.Debug("Mining a new block")

			stoppedEarly, blockReceivedEarly := consensus.MineBlock(currentBlock, stopMiningChanCleaned)
			if stoppedEarly {
				if !reflect.DeepEqual(blockReceivedEarly, block.Block{}) {
					log.Info("Mining stopped early because synchronization is required")
					log.Info("Waiting for the blockchain to be synchronized with the network")
					log.Info("Putting the transactions back in the transaction pool")
					trxPool = append(trxPool, currentBlock.Transactions...)
					continue
				}
				log.Info("Mining stopped early because of a new block received")
				lastBlockStored = block.Block{} // Reset the last block stored and be sure its not nil to avoid errors with copier
				err := copier.Copy(&lastBlockStored, blockReceivedEarly)
				if err != nil {
					log.Errorln("Error copying the block received early : ", err)
				}

				// If there are transactions that are in my current block but not in the received block
				// I need to put them back in the transaction pool to be sure they are mined
				// We can simply compare the merkle root of the two blocks to know if there are transactions that are not in the received block
				if !bytes.Equal(currentBlock.MerkleRoot, blockReceivedEarly.MerkleRoot) {
					log.Debug("Some transactions in the current block are not in the received block")
					// We can simply use TransactionID to know which transactions are not in the received block
					// We can then put them back in the transaction pool
					currentBlockTransactionIDs := currentBlock.GetTransactionIDsMap()
					blockReceivedEarlyTransactionIDs := blockReceivedEarly.GetTransactionIDsMap()
					for trxID, trxData := range currentBlockTransactionIDs {
						if _, ok := blockReceivedEarlyTransactionIDs[trxID]; !ok {
							trxPool = append(trxPool, trxData)
						}
					}
				}
				continue
			}

			err = currentBlock.SignBlock(ecdsaKeyPair)
			if err != nil {
				log.Errorln("Error signing the block : ", err)
				continue
			}

			currentBlockHashEncoded := fmt.Sprintf("%x", block.ComputeHash(*currentBlock))
			log.Infoln("Current block hash : ", currentBlockHashEncoded, " TIMESTAMP : ", currentBlock.Timestamp)
			if !reflect.DeepEqual(previousBlock, &block.Block{}) {
				previousBlockHashEncoded := fmt.Sprintf("%x", block.ComputeHash(*previousBlock))
				log.Infoln("Previous block hash : ", previousBlockHashEncoded, " TIMESTAMP : ", previousBlock.Timestamp)
			}
			if err := blockValidator.Validate(*currentBlock, *previousBlock); err != nil {
				log.Warn("Block is invalid")
				// Return transactions of the current block to the transaction pool
				trxPool = append(trxPool, currentBlock.Transactions...)
				continue
			}

			serializedBlock, err := currentBlock.Serialize()
			if err != nil {
				log.Errorln("Error serializing the block : ", err)
				continue
			}
			if err = blockAnnouncementTopic.Publish(ctx, serializedBlock); err != nil {
				log.Errorln("Error publishing the block : ", err)
				continue
			}

			log.Infof("Block mined and published with hash %v at height %d", block.ComputeHash(*currentBlock), currentBlock.Height)

			// To be sure we retrieve the last block stored
			for {
				lastBlockStored, err = blockchainDB.GetLastBlock()
				if err != nil {
					log.Errorln("Error getting the last block stored : ", err)
				}

				if !reflect.DeepEqual(lastBlockStored, block.Block{}) && lastBlockStored.Height >= currentBlock.Height {
					break
				}
				time.Sleep(5 * time.Second)
			}
		}
	}()

	/*
	* DISPLAY PEER CONNECTEDNESS CHANGES
	 */
	// Subscribe to EvtPeerConnectednessChanged events
	subNet, err := host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Errorln("Failed to subscribe to EvtPeerConnectednessChanged:", err)
	}
	defer subNet.Close()

	// Handle connection events in a separate goroutine
	go func() {
		for e := range subNet.Out() {
			evt := e.(event.EvtPeerConnectednessChanged)
			if evt.Connectedness == network.Connected {
				log.Debugln("Peer connected:", evt.Peer)
			} else if evt.Connectedness == network.NotConnected {
				log.Debugln("Peer disconnected:", evt.Peer)
			}
		}
	}()

	utils.WaitForTermSignal()
}
