package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/fullnode"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/internal/pebble"
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

	// // Join Blacklist topic
	// blacklistTopic, err := ps.Join(cfg.BlacklistStringFlag)
	// if err != nil {
	// 	log.Panicf("Failed to join blacklist topic : %s\n", err)
	// }

	// Waiting to receive blocks from the minor nodes
	go func() {
		// Check if the node has a blockchain
		hasBlockchain := pebble.HasABlockchain()

		for {
			var isPrevBlockStored bool
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

			// Check if the block received is the genesis block
			genesisBlock := fullnode.IsGenesisBlock(blockAnnounced)

			// If the node doesn't have a blockchain and the block received is the genesis block
			if !hasBlockchain && genesisBlock {
				log.Debugln("Blockchain doesn't exist and received genesis block")

				// Start a new blockchain
				blockchain, err := pebble.NewBlockchainDB("blockchain")
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
				log.Debugln("Block validate and added to the blockchain : ", proceed)

				blockPublished, err := fullnode.PublishBlockToNetwork(ctx, blockAnnounced, fullNodeAnnouncementTopic)
				if err != nil {
					log.Debugln("Error publishing block to the network : %s\n", err)
					continue
				}
				log.Debugln("Publishing block to the network :", blockPublished)

				// Send the block to IPFS
				if err := fullnode.PublishBlockToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, blockAnnounced); err != nil {
					log.Debugf("error adding the block to IPFS : %s\n", err)
					continue
				}
			}

			// If the node has a blockchain
			if hasBlockchain {

				// Get the blockchain from the database
				blockchain, err := pebble.NewBlockchainDB("blockchain")
				if err != nil {
					log.Debugln("Error opening the blockchain database : %s\n", err)
					continue
				}

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
					log.Debugln("Block validate and added to the blockchain : ", proceed)

					blockPublished, err := fullnode.PublishBlockToNetwork(ctx, blockAnnounced, fullNodeAnnouncementTopic)
					if err != nil {
						log.Debugln("Error publishing block to the network : %s\n", err)
						continue
					}
					log.Debugln("Publishing block to the network :", blockPublished)

					// Send the block to IPFS
					if err := fullnode.PublishBlockToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, blockAnnounced); err != nil {
						log.Debugf("error adding the block to IPFS : %s\n", err)
						continue
					}
				}
			}

			// If the previous block is not stored in the database and the block is not the genesis block
			if !isPrevBlockStored || !genesisBlock {

				for { // Add the receive block in a list
					blockReceive[blockAnnounced.Timestamp] = blockAnnounced
					// Artificial delay to allow for more blocks to arrive.

					for {

						// A TESTER FORT // RETURN THE SENDER AND THE CID OF THE BLOCKCHAIN
						// Download the blockchain from the network
						downloaded, err, _ := fullnode.DownloadBlockchain(ctx, ipfsAPI, ps)
						if err != nil {
							log.Debugf("error downloading the blockchain : %s\n", err)
							time.Sleep(time.Second * 5)
							continue
						}

						if !downloaded {
							log.Debugln("Blockchain not downloaded")
							continue // not downloaded so we try again
						}

						// Get the blockchain from the database
						blockchain, err := pebble.NewBlockchainDB("blockchain")
						if err != nil {
							log.Debugln("Error opening the blockchain database : %s\n", err)
							continue
						}

						lastBlock := blockchain.GetLastBlock()

						// Check the integrity of the blockchain downloaded
						integrity, err := blockchain.VerifyBlockchainIntegrity(lastBlock)
						if err != nil {
							log.Debugf("error verifying blockchain integrity : %s\n", err)
							continue
						}

						if !integrity {
							// // Blacklist the sender and the cid of the blockchain
							// blacklistMessage := []byte(sender + " " + cidBlockChain.String())
							// if err = blacklistTopic.Publish(ctx, blacklistMessage); err != nil {
							// 	log.Debugln("Error publishing blacklist message to the network : ", err)
							// }
							continue
						}
						break // the blockchain is verified
					}

					// Get the blockchain from the database
					blockchain, err := pebble.NewBlockchainDB("blockchain")
					if err != nil {
						log.Debugln("Error opening the blockchain database : %s\n", err)
						continue
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
						//  Delete the blockchain and wait for the next block
						if err = os.RemoveAll("blockchain"); err != nil {
							log.Debugf("error deleting the blockchain : %s\n", err)
							continue
						}
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

				blockchain, err := pebble.NewBlockchainDB("blockchain")
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
				added, message := pebble.AddBlockToBlockchain(nextBlock, blockchain)
				if !added {
					log.Debugln(message)
				}
				log.Debugln(message)

				published, err := fullnode.PublishBlockToNetwork(ctx, nextBlock, fullNodeAnnouncementTopic)
				if err != nil {
					log.Debugln("Error publishing block to the network : %s\n", err)
					continue
				}
				log.Debugln("Publishing block to the network :", published)

				// Send the block to IPFS
				if err := fullnode.PublishBlockToIPFS(ctx, cfg, nodeIpfs, ipfsAPI, nextBlock); err != nil {
					log.Debugf("error adding the block to IPFS : %s\n", err)
					continue
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

	// // Join the topic to receive the blockchain
	// fullNodeGivingBlockchainTopic, err := ps.Join("FullNodeGivingBlockchain")
	// if err != nil {
	// 	log.Debugln("Error joining to FullNodeGivingBlockchain topic : ", err)
	// }
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

			// // Answer to the request with the last CID of the blockchain
			// if cidBlockChain.RootCid().Defined() {
			// 	err := fullNodeGivingBlockchainTopic.Publish(ctx, []byte(cidBlockChain.String()))
			// 	if err != nil {
			// 		log.Debugln("failed to publish latest blockchain CID: %s", err)
			// 	}
			// }
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
