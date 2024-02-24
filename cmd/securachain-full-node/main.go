package main

import (
	"context"
	"flag"
	"os"
	"sort"

	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
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
			blockReceive := make(map[int64][]*block.Block)

			

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

				// Serialize the block
				blockBytes, err := blockAnnounced.Serialize()
				if err != nil {
					log.Debugln("error serializing block : %s\n", err)
					continue
				}

				// Publish the block to the network (for minors and indexing and searching nodes and storage nodes)
				log.Debugln("Publishing block announcement to the network :", string(blockBytes))

				if err = fullNodeAnnouncementTopic.Publish(ctx, blockBytes); err != nil {
					log.Debugln("Error publishing block announcement to the network : ", err)
					continue
				}

				// Send the blockchain to IPFS
				newCidBlockChain, err = fullnode.AddBlockchainToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, cidBlockChain)
				if err != nil {
					log.Debugf("error adding the blockchain to IPFS : %s\n", err)
					continue
				}
				cidBlockChain = newCidBlockChain
				newCidBlockChain = path.ImmutablePath{}
			}

			// If the node has a blockchain
			if hasBlockchain {
				// Verify if the previous block is stored in the database
				// Deserialize the previous block
				prevBlock, err := block.DeserializeBlock(blockAnnounced.PrevBlock)
				if err != nil {
					log.Debugln("error deserializing previous block : %s\n", err)
					continue
				}

				// Verify if the previous block is stored in the database
				blockchain := databaseInstance

				isPrevBlockStored, err := blockchain.IsIn(prevBlock)
				if err != nil {
					log.Debugln("error checking if previous block is in the blockchain : %s\n", err)
					continue
				}

			}


			/*
			* Validate blocks coming from minors
			* Manage conflicts
			*/

			blockBuff := make(map[int64][]*block.Block)

			listOfBlocks, err := fullnode.HandleIncomingBlock(blockAnnounced, blockBuff, databaseInstance)
			if err != nil {
				log.Debugf("error handling incoming block : %s\n", err)
				continue
			}

			if len(listOfBlocks) > 1 {
				// Serialize the list of blocks
				listOfBlocksBytes, err := json.Marshal(listOfBlocks)
				if err != nil {
					log.Debugf("error serializing list of blocks : %s\n", err)
					continue
				}

				// Return all blocks with the same timestamp for the minor node to select based on the longest chain
				if err := MinorConflictsTopic.Publish(ctx, listOfBlocksBytes); err != nil {
					log.Debugf("error publishing blocks with the same timestamp to the minor : %s\n", err)
					continue
				}
			}

			blockAnnounced = listOfBlocks[0]


			/*
			* Is the node has a blockchain ?
			 */

			if fullnode.HasABlockchain() {
				log.Debugln("Blockchain exist")

				// Deserialize the previous block
				prevBlock, err := block.DeserializeBlock(blockAnnounced.PrevBlock)
				if err != nil {
					log.Debugln("error deserializing previous block : %s\n", err)
					continue
				}

				// Verify if the previous block is stored in the database
				blockchain := databaseInstance

				isPrevBlockStored, err := blockchain.IsIn(prevBlock)
				if err != nil {
					log.Debugln("error checking if previous block is in the blockchain : %s\n", err)
					continue
				}

				if isPrevBlockStored {
					log.Debugln("Previous block is in the database")

					// Add the block to the blockchain
					action, message := fullnode.AddBlockToBlockchain(blockchain, blockAnnounced)
					if !action {
						log.Debugln(message)
						continue
					}

					// Serialize the block
					blockBytes, err := blockAnnounced.Serialize()
					if err != nil {
						log.Debugln("error serializing block : %s\n", err)
						continue
					}

					// Publish the block to the network (for minors and indexing and searching nodes and storage nodes)
					log.Debugln("Publishing block announcement to the network :", string(blockBytes))

					if err = fullNodeAnnouncementTopic.Publish(ctx, blockBytes); err != nil {
						log.Debugln("Error publishing block announcement to the network : ", err)
						continue
					}

					// Send the blockchain to IPFS
					newCidBlockChain, err = fullnode.AddBlockchainToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, cidBlockChain)
					if err != nil {
						log.Debugf("error adding the blockchain to IPFS : %s\n", err)
						continue
					}
					cidBlockChain = newCidBlockChain
					newCidBlockChain = path.ImmutablePath{}
				}
				continue
			} else {
				log.Debugln("Blockchain doesn't exist")

				// If the block is the genesis block, start a new blockchain
				if blockAnnounced.PrevBlock == nil {
					log.Debugln("Received genesis block")
					isGenesisBlock = true

					// Start a new blockchain
					blockchain, err := pebble.NewPebbleTransactionDB("blockchain")
					if err != nil {
						log.Debugln("Error creating a new blockchain database : %s\n", err)
						continue
					}

					// Add the block to the blockchain
					action, message := fullnode.AddBlockToBlockchain(blockchain, blockAnnounced)
					if !action {
						log.Debug(message)
						continue
					}

					// Serialize the block
					blockBytes, err := blockAnnounced.Serialize()
					if err != nil {
						log.Debugln("error serializing block : %s\n", err)
						continue
					}

					// Publish the block to the network (for minors and indexing and searching nodes and storage nodes)
					log.Debugln("Publishing block announcement to the network :", string(blockBytes))

					if err = fullNodeAnnouncementTopic.Publish(ctx, blockBytes); err != nil {
						log.Debugln("Error publishing block announcement to the network : ", err)
						continue
					}

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

			if !isPrevBlockStored || !isGenesisBlock {
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

				// Sort the blocks by their timestamp
				sort.SliceStable(blockReceive[blockAnnounced.Timestamp], func(i, j int) bool {
					return uint32(blockReceive[blockAnnounced.Timestamp][i].Timestamp) < uint32(blockReceive[blockAnnounced.Timestamp][j].Timestamp)
				})

				// Comparer la blockchain avec la liste des blocks en attentes
				for _, blocks := range blockReceive {
					isInBlockchain, err := blockchain.IsIn(blocks)
					if err != nil {
						log.Debugf("error checking if block is in the blockchain : %s\n", err)
						continue
					}

					if isInBlockchain() {
						continue // The block is already in the blockchain, go for the next one
					}

					blockchainIntegrity, err := blockchain.VerifyBlockchainIntegrity(block)
					if err != nil {
						log.Debugf("error verifying blockchain integrity : %s\n", err)
						continue
					}

					// Add the block to the blockchain
					action, message := fullnode.AddBlockToBlockchain(blockchain, block)
					if !action {
						log.Debugln(message)
					}
				}

				if blockchainIntegrity {
					// Add the block to the blockchain
					action, message := fullnode.AddBlockToBlockchain(blockchain, block)
					if !action {
						log.Debugln(message)
					}
				} else {
					//  Delete the blockchain and wait for the next block
					if err = os.RemoveAll("blockchain"); err != nil {
						log.Debugf("error deleting the blockchain : %s\n", err)
						continue
					}
					break // should quit the loop and go back to the first if
				}

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
							blockAnnounced, err := block.DeserializeBlock(msg.Data)
							if err != nil {
								log.Debugln("Error deserializing block announcement : %s", err)
								continue
							}

							// verify the integrity of the blockchain
							blockchainIntegrity, err := blockchain.VerifyBlockchainIntegrity(blockAnnounced)
							if err != nil {
								log.Debugf("error verifying blockchain integrity : %s\n", err)
								continue
							}
							if !blockchainIntegrity {
								// If the blockchain is not verify
								// Forget this block and wait for the next one
								continue
							}
							chanBlock <- blockAnnounced
						}
					}()

					newBlock := <-chanBlock

					// Add the block to the blockchain
					action, message = fullnode.AddBlockToBlockchain(blockchain, newBlock)
					if !action {
						log.Debugln(message)
					}

					// Serialize the block
					newBlockBytes, err := newBlock.Serialize()
					if err != nil {
						log.Debugf("error serializing block : %s\n", err)
						continue
					}

					// Publish the block to the network (for minors and indexing and searching nodes and storage nodes)
					log.Debugln("Publishing block announcement to the network :", string(newBlockBytes))

					if err = fullNodeAnnouncementTopic.Publish(ctx, newBlockBytes); err != nil {
						log.Debugln("Error publishing block announcement to the network : ", err)
					}

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
