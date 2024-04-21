package main

import (
	"context"
	"flag"
	"fmt"
	"slices"
	"time"

	"github.com/jinzhu/copier"
	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
)

var (
	yamlConfigFilePath     = flag.String("config", "", "Path to the yaml config file")
	generateKeys           = flag.Bool("genKeys", false, "Generate new ECDSA and AES keys to the paths specified in the config file")
	requiresSync           = false
	requiresPostSync       = false
	blockProcessingEnabled = true
	pendingBlocks          = []*block.Block{}
	trxPool                = []transaction.Transaction{}
	stopMiningChan         = make(chan consensus.StopMiningSignal)
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

	if *generateKeys {
		node.GenerateKeys(cfg, log)
	}

	ecdsaKeyPair, _ := node.LoadKeys(cfg, log)

	// For the mining process
	var previousBlock *block.Block = nil
	currentBlock := block.NewBlock(trxPool, nil, 1, ecdsaKeyPair)

	/*
	* IPFS NODE
	 */
	ipfsAPI, nodeIpfs := node.InitializeIPFSNode(ctx, cfg, log)

	/*
	* IPFS MEMORY MANAGEMENT
	 */
	storageMax, err := ipfs.ChangeStorageMax(nodeIpfs, cfg.MemorySpace)
	if err != nil {
		log.Warnf("Failed to change storage max: %s", err)
	}
	log.Debugf("Storage max changed: %v", storageMax)

	freeMemorySpace, err := ipfs.FreeMemoryAvailable(ctx, nodeIpfs)
	if err != nil {
		log.Warnf("Failed to get free memory available: %s", err)
	}
	log.Debugf("Free memory available: %v", freeMemorySpace)

	memoryUsedGB, err := ipfs.MemoryUsed(ctx, nodeIpfs)
	if err != nil {
		log.Warnf("Failed to get memory used: %s", err)
	}
	log.Debugf("Memory used: %v", memoryUsedGB)

	/*
	* BLOCKCHAIN DATABASE
	 */
	blockchain, err := blockchaindb.NewBlockchainDB(log, "blockchain")
	if err != nil {
		log.Debugln("Error creating or opening a database : %s\n", err)
	}

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
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// KeepRelayConnectionAlive
	node.PubsubKeepRelayConnectionAlive(ctx, ps, host, cfg, log)

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

	// Service 1 : Receiption of blocks
	blockReceieved := make(chan *block.Block, 15)
	go func() {
		for {
			blockAnnounced, err := fullnode.ReceiveBlock(log, ctx, subBlockAnnouncement)
			if err != nil {
				log.Debugln("Error waiting for the next block : %s\n", err)
				continue
			}

			// If the node is synchronizing with the network
			if requiresSync {
				pendingBlocks = append(pendingBlocks, blockAnnounced)
			} else {
				blockReceieved <- blockAnnounced
			}
		}
	}()

	// Service 2: Handle the block received
	go func() {
		for {
			if !blockProcessingEnabled {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case bReceive := <-blockReceieved:
				log.Debugln("Normal block processing")
				log.Debugln("State : block processing enabled ", blockProcessingEnabled)

				// 1 . Validation of the block
				if block.IsGenesisBlock(bReceive) {
					log.Debugln("Genesis block")
					if !consensus.ValidateBlock(bReceive, nil) {
						log.Debugln("Genesis block is invalid")
						continue
					}
					log.Debugln("Genesis block is valid")
				} else {
					isPrevBlockStored, err := fullnode.PrevBlockStored(log, bReceive, blockchain)
					if err != nil {
						log.Debugln("error checking if previous block is stored : %s", err)
					}

					if !isPrevBlockStored {
						pendingBlocks = append(pendingBlocks, bReceive)
						blockProcessingEnabled = false
						requiresSync = true // This will call the process of synchronizing the blockchain with the network
						continue
					}

					prevBlock, err := blockchain.GetBlock(log, bReceive.PrevBlock)
					if err != nil {
						log.Debugln("Error getting the previous block : %s\n", err)
					}

					if !consensus.ValidateBlock(bReceive, prevBlock) {
						log.Debugln("Block is invalid")
						continue
					}
				}

				// 2 . Add the block to the blockchain
				added := blockchaindb.AddBlockToBlockchain(log, bReceive, blockchain)
				if !added {
					continue
				}

				// 3 . Verify the integrity of the blockchain
				if !blockchain.VerifyIntegrity(log) {
					blockProcessingEnabled = false
					requiresSync = true // This will call the process of synchronizing the blockchain with the network
					continue
				}

				// 4 . Add the block transaction to the registry
				if !fullnode.AddBlockTransactionToRegistry(log, cfg, bReceive) {
					log.Debugln("Error adding the block transactions to the registry")
				}

				// 5 . Send the block to IPFS
				if !ipfs.PublishBlock(log, ctx, cfg, nodeIpfs, ipfsAPI, bReceive) {
					log.Debugln("Error publishing the block to IPFS")
				}

				bReceivedHash := fmt.Sprintf("%x", block.ComputeHash(bReceive))
				bReceivedMinerAddress := fmt.Sprintf("%x", bReceive.MinerAddr)
				log.Warnln("[NEW BLOCK MINED] Block at height ", bReceive.Height, " with hash ", bReceivedHash, " mined by ", bReceivedMinerAddress, " at ", bReceive.Timestamp, " with ", len(bReceive.Transactions), " transactions stored in the blockchain")

				// 6 . Stop the mining process if a new block with the same height or higher is received
				// Conflict is implicitely resolved here.
				// For example : Two miners mine a block at the same height. The first block received will be stored in the blockchain
				if bReceive.Height >= currentBlock.Height {
					log.Warnln("[NEW BLOCK MINED] Block at the same height or higher than the current block so stopping the mining process")
					stopMiningChan <- consensus.StopMiningSignal{Stop: true, BlockReceived: bReceive}
				}
			}
		}
	}()

	// Service 3 : Syncronization
	go func() {
		// create a list of blacklisted nodes
		blackListNode := []string{}
		for {
			if !requiresSync {
				continue
			}
			log.Debugln("Synchronizing the blockchain with the network")
			log.Debugln("State : requires sync ", requiresSync)

			//1 . Ask for a registry of the blockchain
			registryBytes, senderID, err := fullnode.AskForBlockchainRegistry(log, ctx, askingBlockchainTopic, subReceiveBlockchain)
			if err != nil {
				log.Debugln("Error asking the blockchain registry : %s\n", err)
				continue
			}

			// 1.1 Check if the sender is blacklisted
			if slices.Contains(blackListNode, senderID) {
				log.Debugln("Node blacklisted")
				continue
			}

			log.Debugln("Node not blacklisted")

			// 1.2 black list the sender
			blackListNode = append(blackListNode, senderID)
			log.Debugln("Node added to the black list")

			// 2 . Dowlnoad the missing blocks
			downloaded, listOfMissingBlocks, err := fullnode.DownloadMissingBlocks(log, ctx, ipfsAPI, registryBytes, blockchain)
			if err != nil {
				log.Debugln("Error downloading missing blocks : %s\n", err)
			}

			if !downloaded {
				log.Debugln("Blocks not downloaded")
				continue
			}

			// 3 . Valid the downloaded blocks
			for _, b := range listOfMissingBlocks {
				if block.IsGenesisBlock(b) {
					if !consensus.ValidateBlock(b, nil) {
						log.Debugln("Genesis block is invalid")
						continue
					}
					log.Debugln("Genesis block is valid")
				} else {
					prevBlock, err := blockchain.GetBlock(log, b.PrevBlock)
					if err != nil {
						log.Debugln("Error getting the previous block : %s\n", err)
					}
					if !consensus.ValidateBlock(b, prevBlock) {
						log.Debugln("Block is invalid")
						continue
					}
					log.Debugln(b.Height, " is valid")
				}

				// 4 . Add the block to the blockchain
				added := blockchaindb.AddBlockToBlockchain(log, b, blockchain)
				if !added {
					continue
				}

				// 5 . Verify the integrity of the blockchain
				if !blockchain.VerifyIntegrity(log) {
					log.Debugln("Blockchain is not verified")
					continue
				}

				// 6 . Add the block transaction to the registry
				if !fullnode.AddBlockTransactionToRegistry(log, cfg, b) {
					log.Debugln("Error adding the block transactions to the registry")
				}

				// 7 . Send the block to IPFS
				if !ipfs.PublishBlock(log, ctx, cfg, nodeIpfs, ipfsAPI, b) {
					log.Debugln("Error publishing the block to IPFS")
				}
			}

			// 7 . Clear the pending blocks
			blackListNode = []string{}

			// 8 . Change the state of the node
			requiresSync = false
			requiresPostSync = true
			log.Debugln("Blockchain synchronized with the network")
		}
	}()

	// Service 4 : Post-syncronization
	go func() {
		for {
			if !requiresPostSync {
				continue
			}

			log.Debugln("Post-syncronization of the blockchain with the network")
			log.Debugln("State : Post-syncronization ", requiresPostSync)

			// 1. Sort the waiting list by height of the block
			sortedList := fullnode.SortBlockByHeight(log, pendingBlocks)

			for _, b := range sortedList {
				// 2 . Verify if the previous block is stored in the database
				isPrevBlockStored, err := fullnode.PrevBlockStored(log, b, blockchain)
				if err != nil {
					log.Debugln("Error checking if previous block is stored : %s", err)
				}

				if !isPrevBlockStored {
					pendingBlocks = []*block.Block{}
					requiresPostSync = false
					requiresSync = true // This will call the process of synchronizing the blockchain with the network
					continue
				}

				// 3 . Validation of the block
				if block.IsGenesisBlock(b) {
					if !consensus.ValidateBlock(b, nil) {
						log.Debugln("Genesis block is invalid")
						break
					}
					log.Debugln("Genesis block is valid")
				} else {
					prevBlock, err := blockchain.GetBlock(log, b.PrevBlock)
					if err != nil {
						log.Debugln("Error getting the previous block : %s\n", err)
					}

					if !consensus.ValidateBlock(b, prevBlock) {
						log.Debugln("Block is invalid")
						break
					}
					log.Debugln(b.Height, " is valid")
				}

				// 4 . Add the block to the blockchain
				added := blockchaindb.AddBlockToBlockchain(log, b, blockchain)
				if !added {
					continue
				}

				if !blockchain.VerifyIntegrity(log) {
					pendingBlocks = []*block.Block{}
					requiresPostSync = false
					requiresSync = true // This will call the process of synchronizing the blockchain with the network
					continue
				}

				// 5 . Add the block transaction to the registry
				if !fullnode.AddBlockTransactionToRegistry(log, cfg, b) {
					log.Debugln("Error adding the block transactions to the registry")
				}

				// 6 . Send the block to IPFS
				if !ipfs.PublishBlock(log, ctx, cfg, nodeIpfs, ipfsAPI, b) {
					log.Debugln("Error publishing the block to IPFS")
				}
			}
			// 6 . Change the state of the node
			requiresPostSync = false
			blockProcessingEnabled = true
			log.Debugln("Post-syncronization done")
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
			if !fullnode.SendBlocksRegistryToNetwork(log, ctx, cfg, receiveBlockchainTopic) {
				log.Debugln("Error sending the registry of the blockchain")
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
			if !fullnode.SendOwnersFiles(log, ctx, cfg, ownerAddressStr, sendFilesTopic) {
				log.Debugln("Error sending the files of the owner")
				continue
			}
		}
	}()

	// Handle the transactions received from the storage nodes
	go func() {
		log.Debug("Waiting for 10 seconds before starting the transaction processing")
		time.Sleep(30 * time.Second)
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
			if !consensus.ValidateTransaction(trx) {
				log.Warn("Transaction is invalid")
				continue
			}
			trxPool = append(trxPool, trx)
			log.Debugf("Transaction pool size : %d", len(trxPool))
		}
	}()

	// Mining block
	go func() {
		log.Debug("Waiting for 10 seconds before starting the mining process")
		time.Sleep(30 * time.Second)
		log.Info("Mining process started")
		lastBlockStored := blockchain.GetLastBlock(log)
		for {
			// The mining process requires the blockchain to be up to date
			if !blockProcessingEnabled || requiresSync || requiresPostSync {
				continue
			}
			if lastBlockStored != nil {
				previousBlock = lastBlockStored
				previousBlockHash := block.ComputeHash(previousBlock)
				log.Infof("Last block stored on chain : %v at height %d", previousBlockHash, previousBlock.Height)
				currentBlock.PrevBlock = previousBlockHash
				currentBlock.Height = previousBlock.Height + 1
				currentBlock.Transactions = trxPool
				trxPool = []transaction.Transaction{}
			} else {
				log.Infof("No block stored on chain")
			}

			log.Debug("Mining a new block")

			currentBlock.MerkleRoot = currentBlock.ComputeMerkleRoot()

			stoppedEarly, blockReceivedEarly := consensus.MineBlock(currentBlock, stopMiningChan)
			if stoppedEarly {
				log.Info("Mining stopped early because of a new block received")
				lastBlockStored = &block.Block{} // Reset the last block stored and be sure its not nil to avoid errors with copier
				err := copier.Copy(lastBlockStored, blockReceivedEarly)
				if err != nil {
					log.Errorln("Error copying the block received early : ", err)
				}
				continue
			}

			err = currentBlock.SignBlock(ecdsaKeyPair)
			if err != nil {
				log.Errorln("Error signing the block : ", err)
				continue
			}

			log.Infoln("Current block hash : ", block.ComputeHash(currentBlock))
			if previousBlock != nil {
				log.Infoln("Previous block hash : ", block.ComputeHash(previousBlock))
			}
			if !consensus.ValidateBlock(currentBlock, previousBlock) {
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

			log.Infof("Block mined and published with hash %v at height %d", block.ComputeHash(currentBlock), currentBlock.Height)

			// To be sure we retrieve the last block stored
			for {
				log.Debug("Waiting for the last block stored to be updated and retrieved")
				lastBlockStored = blockchain.GetLastBlock(log)
				if lastBlockStored != nil && lastBlockStored.Height >= currentBlock.Height {
					break
				}
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
