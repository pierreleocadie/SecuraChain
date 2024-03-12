package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/blockchaindb"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
)

var yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")
var needSync = false

func main() {
	log := ipfsLog.Logger("full-node")
	if err := ipfsLog.SetLogLevel("full-node", "DEBUG"); err != nil {
		log.Errorln("Failed to set log level : ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flag.Parse()

	// Load the config file
	if *yamlConfigFilePath == "" {
		log.Errorln("Please provide a path to the yaml config file")
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(*yamlConfigFilePath)
	if err != nil {
		log.Errorln("Error loading config file : ", err)
		os.Exit(1)
	}

	/*
	* IPFS NODE
	 */
	ipfsAPI, nodeIpfs := node.InitializeIPFSNode(ctx, cfg, log)
	// dhtAPI := ipfsAPI.Dht()

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

	blockchain, err := blockchaindb.NewBlockchainDB("blockchain")
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
	node.SetupDHTDiscovery(ctx, cfg, host, false)

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
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

	// Waiting to receive blocks from the minor nodes
	go func() {
		for {
			blockReceive := make(map[int64]*block.Block)

			blockAnnounced, err := fullnode.ReceiveBlock(ctx, subBlockAnnouncement)
			if err != nil {
				log.Debugln("Error waiting for the next block : %s\n", err)
				continue
			}

			// validty of the block before adding it to the buffer
			//timestamp +- 1 sec
			//go rountine
			// +- 5 sec le timestamp actuel si oui on ajoute le block au buffer
			// si on recoit un seul block on switch sur la wiating list instead of the conccurence liste

			// Handle incoming block
			handleBlock, err := fullnode.HandleIncomingBlock(ctx, cfg, nodeIpfs, ipfsAPI, blockAnnounced, blockchain)
			if err != nil {
				log.Debugln("Error handling incoming block : %s\n", err)
				continue
			}

			// If the previous block is not stored in the database and the block is not the genesis block
			if !handleBlock {

				// Waiting 2sec to add the block received to the buffer
				endTime := time.Now().Add(2 * time.Second)
				for time.Now().Before(endTime) {
					blockReceive[blockAnnounced.Timestamp] = blockAnnounced
					log.Debugln("Block added to the buffer")
				}

				// verify the integrity of the blockchain
				verified, err := blockchain.VerifyBlockchainIntegrity(blockchain.GetLastBlock())
				if err != nil {
					log.Debugln("Error verifying blockchain integrity : %s", err)
					continue
				}

				if !verified {
					log.Debugln("Blockchain is not integrity")
					integrity := blockchaindb.IntegrityAndUpdate(ctx, ipfsAPI, ps, blockchain)
					if !integrity {
						log.Debugln("Blockchain is not integrity")
						continue
					}
				}

				// Compare the blocks received to the blockchain and sort by timestamp
				blockLists := fullnode.CompareBlocksToBlockchain(blockReceive, blockchain)

				// Valid and add the blocks to the blockchain
				for _, b := range blockLists {
					isProcessed, err := fullnode.ProcessBlock(b, blockchain)
					if err != nil {
						log.Debugf("error processing block : %s\n", err)
						continue
					}
					if !isProcessed {
						log.Debugln("Block not processed")
					}
				}

				verified2, err := blockchain.VerifyBlockchainIntegrity(blockchain.GetLastBlock())
				if err != nil {
					log.Debugf("error verifying blockchain integrity : %s\n", err)
					continue
				}

				// If the blockchain isn't verify
				if !verified2 {
					// Download missing blocks
					integrity := blockchaindb.IntegrityAndUpdate(ctx, ipfsAPI, ps, blockchain)
					if !integrity {
						log.Debugln("Blockchain is not integrity")
						continue
					}
				}

				nextBlockReceive := make(chan *block.Block)
				for {
					nextBlock, err := fullnode.ReceiveBlock(ctx, subBlockAnnouncement)
					if err != nil {
						log.Debugln("Error waiting for the next block : %s\n", err)
						continue
					}
					nextBlockReceive <- nextBlock

					validate := consensus.ValidateBlock(nextBlock, blockchain.GetLastBlock())
					if !validate {
						log.Debugln("Block not validated")
						continue
					}

					isVerified, err := blockchain.VerifyBlockchainIntegrity(nextBlock)
					if err != nil {
						log.Debugf("error verifying blockchain integrity : %s\n", err)
						continue
					}

					if !isVerified {
						log.Debugln("Blockchain not verified")
						continue
					}
					break
				}
				nextBlock := <-nextBlockReceive

				handleNextBlock, err := fullnode.HandleIncomingBlock(ctx, cfg, nodeIpfs, ipfsAPI, nextBlock, blockchain)
				if err != nil {
					log.Debugf("error handling incoming block : %s\n", err)
					continue
				}

				if !handleNextBlock {
					log.Debugln("Block not handled")
				}

			}
		}
	}()

	// Service 1 : Receipt of the block from the minor nodes
	blockReceieved := make(chan *block.Block)
	go func() {
		for {
			blockAnnounced, err := fullnode.ReceiveBlock(ctx, subBlockAnnouncement)
			if err != nil {
				log.Debugln("Error waiting for the next block : %s\n", err)
				continue
			}
			blockReceieved <- blockAnnounced
		}
	}()

	// Service 2: Handle the block received
	go func() {

	}()

	// Join the topic to ask for the json file of the blockchain
	askingBlockchainTopic, err := ps.Join(cfg.AskingBlockchainStringFlag)
	if err != nil {
		log.Panicf("Failed to join AskingBlockchain topic : %s\n", err)
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

	// Service 3 : Syncronize the blockchain with the network
	go func() {
		// create a list of blacklisted nodes
		blackListNode := []string{}
		for needSync {
			log.Debugln("Syncronizing the blockchain with the network")

			//1 . Ask fot the registry of the blockchain
			registryBytes, sender, err := blockchaindb.AskTheBlockchainRegistry(ctx, askingBlockchainTopic, subReceiveBlockchain)
			if err != nil {
				log.Debugln("Error asking the blockchain registry : %s\n", err)
				continue
			}

			// 1.1 Check if the sender is blacklisted
			if blockchaindb.NodeBlackListed(blackListNode, sender) {
				continue
			}

			// 2 . Verify the integrity of the actual blockchain
			_, err = blockchain.VerifyBlockchainIntegrity(blockchain.GetLastBlock())
			if err != nil {
				log.Debugln("Error verifying blockchain integrity : %s\n", err)
				continue
			}

			// // 2.1 Verify the integrity of the blockchain with the last block received
			// verified2, err := blockchain.VerifyBlockchainIntegrity()
			// if err != nil {
			// 	log.Debugln("Error verifying blockchain integrity on the last block received: %s\n", err)
			// 	continue
			// }

			// 3 . Dowlnoad the missing blocks
			downloaded, err := blockchaindb.DownloadMissingBlocks(ctx, ipfsAPI, registryBytes, blockchain)
			if err != nil {
			}

			// 3.1 if something append we add the sender to the black list
			if !downloaded {
				blackListNode = append(blackListNode, sender)
				continue
			}

			// 4 . Verify the integrity of the blockchain
			verified3, err := blockchain.VerifyBlockchainIntegrity(blockchain.GetLastBlock())
			if err != nil {
				log.Debugln("Error verifying blockchain integrity : %s\n", err)
			}

			// 4.1 if the blockchain is not verified we add the sender to the black list
			if !verified3 {
				blackListNode = append(blackListNode, sender)
				continue
			}

			// 4.2 if the blockchain is verified, we clear the black list
			blackListNode = []string{}

			// 5 . Change the state of needSync to false and break the loop
			needSync = false
			if !needSync {
				break
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
