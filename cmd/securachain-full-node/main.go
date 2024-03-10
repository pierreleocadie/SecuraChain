package main

import (
	"context"
	"flag"
	"os"

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

			msg, err := subBlockAnnouncement.Next(ctx)
			if err != nil {
				log.Panic("Error getting block announcement message : ", err)
			}

			log.Debugln("Received block announcement message from ", msg.GetFrom().String())
			log.Debugln("Received block : ", msg.Data)

			// Deserialize the block announcement
			blockAnnounced, err := block.DeserializeBlock(msg.Data)
			if err != nil {
				log.Debugln("Error deserializing block announcement : %s", err)
				continue
			}

			// Handle incoming block
			handleBlock, err := fullnode.HandleIncomingBlock(ctx, cfg, nodeIpfs, ipfsAPI, blockAnnounced, blockchain)
			if err != nil {
				log.Debugln("Error handling incoming block : %s\n", err)
				continue
			}

			// If the previous block is not stored in the database and the block is not the genesis block
			if !handleBlock {

				for {
					// Add the receive block in a list
					blockReceive[blockAnnounced.Timestamp] = blockAnnounced

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

					// Compare the blocks received to the blockchain
					blockLists := fullnode.CompareBlocksToBlockchain(blockReceive, blockchain)

					// Verify the validy of the blocks list
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

					lastBlock := blockchain.GetLastBlock()
					integrity, err := blockchain.VerifyBlockchainIntegrity(lastBlock)
					if err != nil {
						log.Debugf("error verifying blockchain integrity : %s\n", err)
						continue
					}

					// If the blockchain isn't verify
					if !integrity {
						// Download missing blocks

					}
					break
				}

				// wait for the next block
				msg, err := subBlockAnnouncement.Next(ctx)
				if err != nil {
					log.Panic("Error getting block announcement message : ", err)
				}

				log.Debugln("Received block announcement message from ", msg.GetFrom().String())
				log.Debugln("Received block message : ", msg.Data)

				// Deserialize the block announcement
				nextBlock, err := block.DeserializeBlock(msg.Data)
				if err != nil {
					log.Debugln("Error deserializing block announcement : %s", err)
					continue
				}

				blockchain, err := blockchaindb.NewBlockchainDB("blockchain")
				if err != nil {
					log.Debugln("Error opening the blockchain database : %s\n", err)
					continue
				}

				lastBlock := blockchain.GetLastBlock()

				validate := consensus.ValidateBlock(nextBlock, lastBlock)
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

				// Proceed to validate and add the block to the blockchain
				added, message := blockchaindb.AddBlockToBlockchain(nextBlock, blockchain)
				if !added {
					log.Debugln(message)
				}
				log.Debugln(message)

				// published, err := fullnode.PublishBlockToNetwork(ctx, nextBlock, fullNodeAnnouncementTopic)
				// if err != nil {
				// 	log.Debugln("Error publishing block to the network : %s\n", err)
				// 	continue
				// }
				// log.Debugln("Publishing block to the network :", published)

				// Send the block to IPFS
				if err := blockchaindb.PublishBlockToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, nextBlock); err != nil {
					log.Debugf("error adding the block to IPFS : %s\n", err)
					continue
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
