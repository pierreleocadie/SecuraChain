package main

import (
	"context"
	"flag"
	"os"
	"slices"
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
	netwrk "github.com/pierreleocadie/SecuraChain/internal/network"
)

var yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")
var needSync = false
var needPostSync = false
var treatBlock = true
var waitingList = []*block.Block{}

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

	// Service 1 : Receiption of blocks
	blockReceieved := make(chan *block.Block, 15)
	go func() {
		for {
			blockAnnounced, err := fullnode.ReceiveBlock(ctx, subBlockAnnouncement)
			if err != nil {
				log.Debugln("Error waiting for the next block : %s\n", err)
				continue
			}

			// Display the block received
			b, err := blockAnnounced.Serialize()
			if err != nil {
				log.Debugln("Error serializing the block : %s\n", err)
			}
			log.Debugln(string(b))

			// If the node is synchronizing with the network
			if needSync {
				waitingList = append(waitingList, blockAnnounced)
			} else {
				blockReceieved <- blockAnnounced
			}
		}
	}()

	// Service 2: Handle the block received
	go func() {
		for {
			if !treatBlock {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case bReceive := <-blockReceieved:
				// 2 . Validation of the block
				if block.IsGenesisBlock(bReceive) {
					log.Debugln("Genesis block")
					if !consensus.ValidateBlock(bReceive, nil) {
						log.Debugln("Genesis block is invalid")
						continue
					}
					log.Debugln("Genesis block is valid")
				} else {
					// 1 . Verify if the previous block is stored in the database
					isPrevBlockStored, err := fullnode.PrevBlockStored(log, bReceive, blockchain)
					if err != nil {
						log.Debugln("error checking if previous block is stored : %s", err)
					}

					if !isPrevBlockStored {
						waitingList = append(waitingList, bReceive)
						treatBlock = false
						needSync = true // This will call the process of synchronizing the blockchain with the network
						continue
					}

					prevBlock, err := blockchain.GetBlock(bReceive.PrevBlock)
					if err != nil {
						log.Debugln("Error getting the previous block : %s\n", err)
					}

					if !consensus.ValidateBlock(bReceive, prevBlock) {
						log.Debugln("Block is invalid")
						continue
					}
				}

				// 3 . Add the block to the blockchain
				added, message := blockchaindb.AddBlockToBlockchain(bReceive, blockchain)
				if !added {
					log.Debugln("Block not added to the blockchain : %s\n", message)
					continue
				}
				log.Debugln(message)

				// 4 .  Verify the integrity of the blockchain
				verified, err := blockchain.VerifyBlockchainIntegrity(blockchain.GetLastBlock())
				if err != nil {
					log.Debugln("Error verifying blockchain integrity : %s\n", err)
				}

				if !verified {
					treatBlock = false
					needSync = true // This will call the process of synchronizing the blockchain with the network
					continue
				}

				// Send the block to IPFS
				if err := blockchaindb.PublishBlockToIPFS(log, ctx, cfg, nodeIpfs, ipfsAPI, bReceive); err != nil {
					log.Debugln("error adding the block to IPFS : %s", err)
				}
			}
		}
	}()

	// Service 3 : Syncronization
	go func() {
		// create a list of blacklisted nodes
		blackListNode := []string{}
		for {
			if !needSync {
				continue
			}

			log.Debugln("Syncronizing the blockchain with the network")

			//1 . Ask fot the registry of the blockchain
			registryBytes, sender, err := blockchaindb.AskTheBlockchainRegistry(ctx, askingBlockchainTopic, subReceiveBlockchain)
			if err != nil {
				log.Debugln("Error asking the blockchain registry : %s\n", err)
				continue
			}

			log.Debugln("Registry received from : ", sender)
			log.Debugln("Registry received : ", string(registryBytes))

			// 1.1 Check if the sender is blacklisted
			if slices.Contains(blackListNode, sender) {
				log.Debugln("Node blacklisted")
				continue
			}

			log.Debugln("Node not blacklisted")

			// 1.2 black list the sender
			blackListNode = append(blackListNode, sender)

			log.Debugln("Node added to the black list")

			// 2 . Dowlnoad the missing blocks
			downloaded, listOfMissingBlocks, err := blockchaindb.DownloadMissingBlocks(log, ctx, ipfsAPI, registryBytes, blockchain)
			if err != nil {
				log.Debugln("Error downloading missing blocks : %s\n", err)
			}

			if !downloaded {
				log.Debugln("Blocks not downloaded")
				continue
			}

			log.Debugln("Blocks downloaded : ", downloaded)
			log.Debugln("List of missing blocks : ", listOfMissingBlocks)

			// // just for the logs
			// log.Debugln("------- List of missing blocks -------")

			// for _, b := range listOfMissingBlocks {
			// 	bBytes, err := b.Serialize()
			// 	if err != nil {
			// 		log.Debugln("Error serializing the block : %s\n", err)
			// 	}
			// 	log.Debugln("Block : ", string(bBytes))
			// }

			// 4 . Valid the downloaded blocks
			for _, b := range listOfMissingBlocks {
				// for the logs
				bBytes, err := b.Serialize()
				if err != nil {
					log.Debugln("Error serializing the block : %s\n", err)
				}
				log.Debugln("---------- Processing missing into validation ------------- : ", string(bBytes))

				if block.IsGenesisBlock(b) {
					if !consensus.ValidateBlock(b, nil) {
						log.Debugln("Genesis block is invalid")
						continue
					}
				} else {
					prevBlock, err := blockchain.GetBlock(b.PrevBlock)
					if err != nil {
						log.Debugln("Error getting the previous block : %s\n", err)
					}
					if !consensus.ValidateBlock(b, prevBlock) {
						log.Debugln("Block is invalid")
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

				// 4.2  Verify the integrity of the blockchain
				verified, err := blockchain.VerifyBlockchainIntegrity(blockchain.GetLastBlock())
				if err != nil {
					log.Debugln("Error verifying blockchain integrity : %s\n", err)
				}

				if !verified {
					log.Debugln("Blockchain is not verified")
					continue
				}

				// Send the block to IPFS
				if err := blockchaindb.PublishBlockToIPFS(log, ctx, cfg, nodeIpfs, ipfsAPI, b); err != nil {
					log.Debugln("error adding the block to IPFS : %s", err)

				}
			}

			// 5. if the blockchain is verified, we clear the black list
			blackListNode = []string{}

			// 6 . Change the states
			needSync = false
			needPostSync = true
			log.Debugln("Blockchain synchronized with the network")
		}
	}()

	// Service 4 : Post-syncronization
	go func() {
		for {
			if !needPostSync {
				continue
			}
			log.Debugln("Post-syncronization of the blockchain with the network")

			// 1. Sort the waiting list by height of the block
			sortedList := fullnode.SortBlockByHeight(log, waitingList)

			for _, b := range sortedList {
				// 2 . Verify if the previous block is stored in the database
				isPrevBlockStored, err := fullnode.PrevBlockStored(log, b, blockchain)
				if err != nil {
					log.Debugln("Error checking if previous block is stored : %s", err)
				}

				if !isPrevBlockStored {
					waitingList = []*block.Block{}
					needPostSync = false
					needSync = true // This will call the process of synchronizing the blockchain with the network
					continue
				}

				// 3 . Validation of the block
				if block.IsGenesisBlock(b) {
					if !consensus.ValidateBlock(b, nil) {
						log.Debugln("Genesis block is invalid")
						break
					}
				} else {
					prevBlock, err := blockchain.GetBlock(b.PrevBlock)
					if err != nil {
						log.Debugln("Error getting the previous block : %s\n", err)
					}

					if !consensus.ValidateBlock(b, prevBlock) {
						log.Debugln("Block is invalid")
						break
					}
				}

				// 4 . Add the block to the blockchain
				added, message := blockchaindb.AddBlockToBlockchain(b, blockchain)
				if !added {
					log.Debugln("Block not added to the blockchain : %s\n", message)
					continue
				}
				log.Debugln(message)

				// 5 .  Verify the integrity of the blockchain
				verified, err := blockchain.VerifyBlockchainIntegrity(blockchain.GetLastBlock())
				if err != nil {
					log.Debugln("Error verifying blockchain integrity : %s\n", err)
				}

				if !verified {
					waitingList = []*block.Block{}
					needPostSync = false
					needSync = true // This will call the process of synchronizing the blockchain with the network
					continue
				}

				// Send the block to IPFS
				if err := blockchaindb.PublishBlockToIPFS(log, ctx, cfg, nodeIpfs, ipfsAPI, b); err != nil {
					log.Debugln("error adding the block to IPFS : %s", err)
				}
			}
			needPostSync = false
			treatBlock = true
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

			// Get the registry of the blockchain
			blockRegistry, err := blockchaindb.ReadBlockDataFromFile(cfg.BlocksRegistryJSON)
			if err != nil {
				log.Debugln("Error reading the registry of the blockchain : ", err)
				continue
			}

			// Convert the registry to bytes
			registryBytes, err := blockchaindb.SerializeRegistry(blockRegistry)
			if err != nil {
				log.Debugln("Error serializing the registry of the blockchain : ", err)
				continue
			}

			// Publish the registry of the blockchain
			if err = receiveBlockchainTopic.Publish(ctx, registryBytes); err != nil {
				log.Debugln("Error publishing the registry of the blockchain : ", err)
				continue
			}
		}
	}()

	// KeepRelayConnectionAlive
	keepRelayConnectionAliveTopic, err := ps.Join("KeepRelayConnectionAlive")
	if err != nil {
		log.Warnf("Failed to join KeepRelayConnectionAlive topic: %s", err)
	}

	// Subscribe to KeepRelayConnectionAlive topic
	subKeepRelayConnectionAlive, err := keepRelayConnectionAliveTopic.Subscribe()
	if err != nil {
		log.Warnf("Failed to subscribe to KeepRelayConnectionAlive topic: %s", err)
	}

	// Handle incoming KeepRelayConnectionAlive messages
	go func() {
		for {
			msg, err := subKeepRelayConnectionAlive.Next(ctx)
			if err != nil {
				log.Errorf("Failed to get next message from KeepRelayConnectionAlive topic: %s", err)
				continue
			}
			if msg.GetFrom().String() == host.ID().String() {
				continue
			}
			log.Debugf("Received KeepRelayConnectionAlive message from %s", msg.GetFrom().String())
			log.Debugf("KeepRelayConnectionAlive: %s", string(msg.Data))
		}
	}()

	// Handle outgoing KeepRelayConnectionAlive messages
	go func() {
		for {
			time.Sleep(cfg.KeepRelayConnectionAliveInterval)
			err := keepRelayConnectionAliveTopic.Publish(ctx, netwrk.GeneratePacket(host.ID()))
			if err != nil {
				log.Errorf("Failed to publish KeepRelayConnectionAlive message: %s", err)
				continue
			}
			log.Debugf("KeepRelayConnectionAlive message sent successfully")
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
