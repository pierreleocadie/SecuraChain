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
var inSync = false
var waitingList = []*block.Block{}
var Synced = false

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

	// Service 1 : Receiption of blocks
	blockReceieved := make(chan *block.Block, 15)
	go func() {
		for {
			blockAnnounced, err := fullnode.ReceiveBlock(ctx, subBlockAnnouncement)
			if err != nil {
				log.Debugln("Error waiting for the next block : %s\n", err)
				continue
			}
			// If the node is synchronizing with the network
			// we add the block to the waiting list instead of the buffer
			if inSync {

				waitingList = append(waitingList, blockAnnounced)
				continue
			}

			blockReceieved <- blockAnnounced
		}
	}()

	// Service 2: Handle the block received
	go func() {
		for {
			if Synced {
				// 1. Sort the waiting list by height of the block
				sortedList := fullnode.SortBlockByHeight(waitingList)

				// 2. Valid, add, publish and verify the blockchain for everyblock in the list
				for _, blockInList := range sortedList {

					handled, err := fullnode.HandleIncomingBlock(ctx, cfg, nodeIpfs, ipfsAPI, blockInList, blockchain)
					if err != nil {
						log.Debugln("Error handling incoming block : %s", err)
					}

					if !handled {
						waitingList = []*block.Block{}
						needSync = true // This will call the process of synchronizing the blockchain with the network
						Synced = false
						break
					}
					verified, err := blockchain.VerifyBlockchainIntegrity(blockchain.GetLastBlock())
					if err != nil {
						log.Debugln("Error verifying blockchain integrity : %s", err)
					}

					if !verified {
						waitingList = []*block.Block{}
						needSync = true // This will call the process of synchronizing the blockchain with the network
						Synced = false
						break
					}
					log.Debugln("Block handle")
					break
				}
				// clear the waiting list
				log.Debugln("All blocks handled")
			} else {
				// Handle incoming block
				handleBlock, err := fullnode.HandleIncomingBlock(ctx, cfg, nodeIpfs, ipfsAPI, blockAnnounced, blockchain)
				if err != nil {
					log.Debugln("Error handling incoming block : %s\n", err)
					continue
				}

				if !handleBlock {
					log.Debugln("Block not handled")
					needSync = true // This will call the process of synchronizing the blockchain with the network
				}

				// As long the blockchain is not sync or is in sync we wait
				for needSync && inSync {
					time.Sleep(10 * time.Second)
					if !needSync && !inSync {
						break
					}
				}
				verified, err := blockchain.VerifyBlockchainIntegrity(blockchain.GetLastBlock())
				if err != nil {
					log.Debugln("Error verifying blockchain integrity : %s", err)
				}
			}
		}
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

	// Service 3 : Syncronization
	go func() {
		// create a list of blacklisted nodes
		blackListNode := []string{}
		for needSync {
			log.Debugln("Syncronizing the blockchain with the network")
			inSync = true

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

			// 3 . Dowlnoad the missing blocks
			downloaded, listOfMissingBlocks, err := blockchaindb.DownloadMissingBlocks(ctx, ipfsAPI, registryBytes, blockchain)
			if err != nil {
				log.Debugln("Error downloading missing blocks : %s\n", err)
			}

			if !downloaded {
				log.Debugln("Blocks not downloaded")
				blackListNode = append(blackListNode, sender)
				continue
			}

			// 4 . Valid the downloaded blocks
			for _, b := range listOfMissingBlocks {
				prevBlock, err := block.DeserializeBlock(b.PrevBlock)
				if err != nil {
					log.Debugln("Error deserializing the previous block : %s\n", err)
				}

				if fullnode.IsGenesisBlock(b) {
					if !consensus.ValidateBlock(b, nil) {
						log.Debugln("Genesis block is invalid")
						blackListNode = append(blackListNode, sender)
						continue
					}
				} else {
					if !consensus.ValidateBlock(b, prevBlock) {
						log.Debugln("Block is invalid")
						blackListNode = append(blackListNode, sender)
						continue
					}
				}

				// 4.1 Add the block to the blockchain
				added, message := blockchaindb.AddBlockToBlockchain(b, blockchain)
				if !added {
					log.Debugln("Block not added to the blockchain : %s\n", message)
					continue
				}
				log.Debugln(message)

				// Send the block to IPFS
				if err := blockchaindb.PublishBlockToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, b); err != nil {
					log.Debugln("error adding the block to IPFS : %s", err)

				}

				// 4.2  Verify the integrity of the blockchain
				verified, err := blockchain.VerifyBlockchainIntegrity(blockchain.GetLastBlock())
				if err != nil {
					log.Debugln("Error verifying blockchain integrity : %s\n", err)
				}

				if !verified {
					log.Debugln("Blockchain is not verified")
					blackListNode = append(blackListNode, sender)
					continue
				}
				break

			}

			// 5. if the blockchain is verified, we clear the black list
			blackListNode = []string{}

			// 6 . Change the states
			needSync = false
			if !needSync {
				inSync = false
				Synced = true
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
