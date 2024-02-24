package main

import (
	"context"
	"flag"
	"os"

	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode/pebble"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
)

var yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")
var databaseInstance *pebble.PebbleTransactionDB
var cidBlockChain path.ImmutablePath
var newCidBlockChain path.ImmutablePath

func main() {
	log := ipfsLog.Logger("full-node")
	ipfsLog.SetLogLevel("full-node", "DEBUG")

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

	// Join the topic FullNodeAnnouncementStringFlag
	fullNodeAnnouncementTopic, err := ps.Join(cfg.FullNodeAnnouncementStringFlag)
	if err != nil {
		log.Panicf("Failed to join full node announcement topic : %s\n", err)
	}

	// Join Blacklist topic
	blacklistTopic, err := ps.Join(cfg.BlacklistStringFlag)
	if err != nil {
		log.Panicf("Failed to join blacklist topic : %s\n", err)
	}

	// Waiting to receive the next block announcement
	go func() {
		// Check if the node has a blockchain
		hasBlockchain := fullnode.HasABlockchain()

		for {
			var isPrevBlockStored bool
			blockReceive := make(map[int64]*block.Block)

			msg, err := subBlockAnnouncement.Next(ctx)
			if err != nil {
				log.Panic("Error getting block announcement message : ", err)
			}

			log.Debugln("Received block announcement message from ", msg.GetFrom().String())
			log.Debugln("Received block announcement message : ", msg.Data)

			// Deserialize the block announcement
			blockAnnounced, err := block.DeserializeBlock(msg.Data)
			if err != nil {
				log.Debugln("Error deserializing block announcement : %s", err)
				continue
			}

			// Verify if the block is the genesis block
			genesisBlock := fullnode.IsGenesisBlock(blockAnnounced)

			if !hasBlockchain && genesisBlock {
				log.Debugln("Blockchain doesn't exist and received genesis block")

				// Start a new blockchain
				blockchain, err := pebble.NewPebbleTransactionDB("blockchain")
				if err != nil {
					log.Debugln("Error creating a new blockchain database : %s\n", err)
					continue
				}

				// Proceed to validate and add the block to the blockchain
				proceed, err := fullnode.ProcessBlock(blockAnnounced, blockchain)
				if err != nil {
					log.Debugln("Error processing block : %s\n", err)
					continue
				}
				log.Debugln("Block processed : ", proceed)

				blockPublished, err := fullnode.PublishBlockToNetwork(ctx, blockAnnounced, fullNodeAnnouncementTopic)
				if err != nil {
					log.Debugln("Error publishing block to the network : %s\n", err)
					continue
				}
				log.Debugln("Publishing block announcement to the network :", blockPublished)

				// Send the blockchain to IPFS
				newCidBlockChain, err = fullnode.PublishBlockchainToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, cidBlockChain)
				if err != nil {
					log.Debugf("error adding the blockchain to IPFS : %s\n", err)
					continue
				}
				cidBlockChain = newCidBlockChain
				newCidBlockChain = path.ImmutablePath{}
			}

			// If the node has a blockchain
			if hasBlockchain {
				blockchain := databaseInstance

				// Verify if the previous block is stored in the database
				isPrevBlockStored, err = fullnode.PrevBlockStored(blockAnnounced, blockchain)
				if err != nil {
					log.Debugln(err)
					continue
				}

				if isPrevBlockStored {
					// Proceed to validate and add the block to the blockchain
					proceed, err := fullnode.ProcessBlock(blockAnnounced, blockchain)
					if err != nil {
						log.Debugln("Error processing block : %s\n", err)
						continue
					}
					log.Debugln("Block processed : ", proceed)

					blockPublished, err := fullnode.PublishBlockToNetwork(ctx, blockAnnounced, fullNodeAnnouncementTopic)
					if err != nil {
						log.Debugln("Error publishing block to the network : %s\n", err)
						continue
					}
					log.Debugln("Publishing block announcement to the network :", blockPublished)

					// Send the blockchain to IPFS
					newCidBlockChain, err = fullnode.AddBlockchainToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, cidBlockChain)
					if err != nil {
						log.Debugf("error adding the blockchain to IPFS : %s\n", err)
						continue
					}
					cidBlockChain = newCidBlockChain
					newCidBlockChain = path.ImmutablePath{}
				}
			}

			// If the previous block is not stored in the database and the block is not the genesis block
			if !isPrevBlockStored || !genesisBlock {
				// Add the receive block in a list
				blockReceive[blockAnnounced.Timestamp] = append(blockReceive[blockAnnounced.Timestamp], blockAnnounced)

				// Download the blockchain from the network
				blockchainDownloaded, err, sender := fullnode.DownloadBlockchain(ctx, ipfsAPI, ps)
				if err != nil {
					log.Debugf("error downloading the blockchain : %s\n", err)
					continue
				}

				blockchain := databaseInstance
				if blockchainDownloaded {
					// Check the integrity of the blockchain downloaded
					blockchainIntegrity, err := blockchain.VerifyBlockchainIntegrity(blockAnnounced)
					if err != nil {
						log.Debugf("error verifying blockchain integrity : %s\n", err)
						continue
					}

					if !blockchainIntegrity {
						// Announce the cid and the sender of the blockchain to the network to blacklist the sender
						blacklistMessage := []byte(sender + " " + cidBlockChain.String())
						if err = blacklistTopic.Publish(ctx, blacklistMessage); err != nil {
							log.Debugln("Error publishing blacklist message to the network : ", err)
						}
						continue // to re download the blockchain
					}
				}

				// Compare the blockchain with the list of blocks received
				blockLists := fullnode.CompareBlocksToBlockchain(blockReceive, blockchain)

				// Verify the validy of the blocks list
				for _, blocks := range blockLists {
					// Verify if the block is valid
					if !consensus.ValidateListBlock(blocks, blockchain) {
						log.Debugln("Block list is invalid")
					}
				}

				// Add the blocks to the blockchain
				for _, blocks := range blockLists {
					added, message := fullnode.AddBlockToBlockchain(blockchain, blocks)
					if !added {
						log.Debugln(message)
					}
				}

				// verify the integrity of the blockchain after adding the blocks
				blockchainIntegrity, err := blockchain.VerifyBlockchainIntegrity(blockLists[len(blockLists)][len(blockLists[len(blockLists)])-1])
				if err != nil {
					log.Debugf("error verifying blockchain integrity : %s\n", err)
					continue
				}

				// If the blockchain isn't verify
				if !blockchainIntegrity {
					//  Delete the blockchain and wait for the next block
					if err = os.RemoveAll("blockchain"); err != nil {
						log.Debugf("error deleting the blockchain : %s\n", err)
						continue
					}
					break // should quit the loop and go back to the first if
				} else {
					// wait for the next block
					chanBlock := make(chan *block.Block)
					go func() {
						for {
							msg, err := subBlockAnnouncement.Next(ctx)
							if err != nil {
								log.Panic("Error getting block announcement message : ", err)
							}

							log.Debugln("Received block announcement message from ", msg.GetFrom().String())
							log.Debugln("Received block announcement message : ", msg.Data)

							// Deserialize the block announcement
							nextBlockAnnounced, err := block.DeserializeBlock(msg.Data)
							if err != nil {
								log.Debugln("Error deserializing block announcement : %s", err)
								continue
							}

							prevNextBlock, err := block.DeserializeBlock(nextBlockAnnounced.PrevBlock)
							if err != nil {
								log.Debugln("Error deserializing previous block : %s", err)
								continue
							}

							nextBlockValid := consensus.ValidateBlock(nextBlockAnnounced, prevNextBlock)

							if blockchain.VerifyBlockchainIntegrity(nextBlockAnnounced) {
								chanBlock <- nextBlockAnnounced
							}
						}
					}()

					newBlock := <-chanBlock
					nextBlockAdded, message := fullnode.AddBlockToBlockchain(blockchain, newBlock)
					if !nextBlockAdded {
						log.Debugln(message)
					}


					published, err := fullnode.PublishBlockToNetwork(ctx, newBlock, fullNodeAnnouncementTopic)
					if err != nil {
						log.Debugln("Error publishing block to the network : %s\n", err)
						continue
					}
					

					if published {
						// Send the blockchain to IPFS
						newCidBlockChain, err = fullnode.AddBlockchainToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, cidBlockChain)
						if err != nil {
							log.Debugf("error adding the blockchain to IPFS : %s\n", err)
							continue
						}
						cidBlockChain = newCidBlockChain
						newCidBlockChain = path.ImmutablePath{}
					}

				}
			}
	}()

	// /*
	// * Validate blocks coming from minors
	// * Manage conflicts
	// */

	// blockBuff := make(map[int64][]*block.Block)

	// listOfBlocks, err := fullnode.HandleIncomingBlock(blockAnnounced, blockBuff, databaseInstance)
	// if err != nil {
	// 	log.Debugf("error handling incoming block : %s\n", err)
	// 	continue
	// }

	// if len(listOfBlocks) > 1 {
	// 	// Serialize the list of blocks
	// 	listOfBlocksBytes, err := json.Marshal(listOfBlocks)
	// 	if err != nil {
	// 		log.Debugf("error serializing list of blocks : %s\n", err)
	// 		continue
	// 	}

	// 	// Return all blocks with the same timestamp for the minor node to select based on the longest chain
	// 	if err := MinorConflictsTopic.Publish(ctx, listOfBlocksBytes); err != nil {
	// 		log.Debugf("error publishing blocks with the same timestamp to the minor : %s\n", err)
	// 		continue
	// 	}
	// }

	// blockAnnounced = listOfBlocks[0]

	// Join the topic to ask for the blockchain
	fullNodeAskingForBlockchainTopic, err := ps.Join("FullNodeAskingForBlockchain")
	if err != nil {
		log.Debugln("error joining FullNodeAskingForBlockchain topic : ", err)
	}

	// Join the topic to ask for the blockchain
	subFullNodeAskingForBlockchain, err := fullNodeAskingForBlockchainTopic.Subscribe()
	if err != nil {
		log.Debugln("error joining FullNodeAskingForBlockchain topic : ", err)
	}

	// Join the topic to receive the blockchain
	fullNodeGivingBlockchainTopic, err := ps.Join("FullNodeGivingBlockchain")
	if err != nil {
		log.Debugln("Error joining to FullNodeGivingBlockchain topic : ", err)
	}
	// Handle incoming asking blockchain messages from full nodes
	go func() {
		for {
			msg, err := subFullNodeAskingForBlockchain.Next(ctx)
			if err != nil {
				log.Debugf("error getting blockchain request message : %s\n", err)
			}

			// Ignore self messages
			if msg.ReceivedFrom == host.ID() {
				continue
			}

			log.Debugln("Blockchain request received")

			// Answer to the request with the last CID of the blockchain
			if cidBlockChain.RootCid().Defined() {
				err := fullNodeGivingBlockchainTopic.Publish(ctx, []byte(cidBlockChain.String()))
				if err != nil {
					log.Debugln("failed to publish latest blockchain CID: %s", err)
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
