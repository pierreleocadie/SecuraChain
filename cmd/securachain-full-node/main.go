package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/discovery"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/internal/pebble"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	"github.com/ipfs/boxo/path"
	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
)

var yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")

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
	// Spawn an IPFS node
	ipfsApi, nodeIpfs, err := ipfs.SpawnNode(ctx)
	if err != nil {
		log.Fatalf("Failed to spawn IPFS node: %s", err)
	}

	log.Debugf("IPFS node spawned with PeerID: %s", nodeIpfs.Identity.String())

	/*
	* NODE LIBP2P
	 */
	// Initialize the node
	host := node.Initialize(*cfg)
	defer host.Close()

	// Setup DHT discovery
	node.SetupDHTDiscovery(ctx, host, false)

	log.Debugf("Minor node initialized with PeerID: %s", host.ID().String())

	/*
	* NETWORK PEER DISCOVERY WITH mDNS
	 */
	mdnsDiscovery := discovery.NewMDNSDiscovery(config.RendezvousStringFlag)

	// Run mDNS
	if err := mdnsDiscovery.Run(host); err != nil {
		log.Fatalf("Failed to run mDNS: %s", err)
		return
	}
	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// Join the topic FullNodeAnnouncementStringFlag
	fullNodeAnnouncementTopic, err := ps.Join(cfg.FullNodeAnnouncementStringFlag)
	if err != nil {
		panic(err)
	}
	subFullNodeAnnouncement, err := fullNodeAnnouncementTopic.Subscribe()
	if err != nil {
		panic(err)
	}

	pebbleDB, err := pebble.NewPebbleTransactionDB("blockchain")
	if err != nil {
		panic(err)
	}

	// Handle incoming full node announcement messages
	go func() {
		for {
			msg, err := subFullNodeAnnouncement.Next(ctx)
			if err != nil {
				panic(err)
			}
			log.Debugln("Received FullNodeAnnouncement message from ", msg.GetFrom().String())
			log.Debugln("Full Node Announcement: ", string(msg.Data))

			// Deserialize the full node announcement
			blockAnnounced, err := block.DeserializeBlock(msg.Data)
			if err != nil {
				log.Errorf("Error deserializing block announcement : %s", err)
				continue
			}

			// Verify if the block is existing in the blockchain
			key := blockAnnounced.Signature
			emptyBlock := block.Block{}
			blockVerif, err := pebbleDB.GetBlock(key)
			if err != nil || blockVerif == &emptyBlock {
				// Add to blockchain
				err = pebbleDB.SaveBlock(blockAnnounced.Signature, blockAnnounced)
				if err != nil {
					panic(err)
				}
				continue
			} else {
				log.Debugln("Block already existing in the blockchain")
				blockPebble, err := pebbleDB.GetBlock(key)
				if err != nil {
					panic(err)
				}
				blockPebbleBytes, err := blockPebble.Serialize()
				if err != nil {
					panic(err)
				}

				log.Debugln("\n\nblock :", string(blockPebbleBytes), "/n/n")

				continue
			}

		}
	}()

	// Join the topic BlockAnnouncementStringFlag
	blockAnnouncementTopic, err := ps.Join(cfg.BlockAnnouncementStringFlag)
	if err != nil {
		panic(err)
	}
	subBlockAnnouncement, err := blockAnnouncementTopic.Subscribe()
	if err != nil {
		panic(err)
	}

	// Handle incoming block announcement messages from minors
	go func() {
		for {

			msg, err := subBlockAnnouncement.Next(ctx)
			if err != nil {
				log.Panic("Error getting block announcement message : ", err)
			}

			log.Debugln("Received block announcement message from ", msg.GetFrom().String())
			log.Debugln("Received block announcement message : ", msg.Data)

			/*
			* BLOCK VALIDATION
			 */

			// Deserialize the block announcement
			blockAnnounced, err := block.DeserializeBlock(msg.Data)
			if err != nil {
				log.Errorf("Error deserializing block announcement : %s", err)
				continue
			}

			// blockPrevBlock, err := block.DeserializeBlock(blockAnnounced.PrevBlock)
			// if err != nil {
			// 	log.Errorf("Error deserializing block announcement : %s", err)
			// 	continue
			// }

			// Validate the block
			if !consensus.ValidateBlock(blockAnnounced, nil) {
				log.Error("Block validation failed")
				continue
			} else {

				// Handle broadcasting the block to the network (for full nodes, minors, and indexing and search nodes)
				go func() {
					for {
						blockAnnouncedBytes, err := blockAnnounced.Serialize()
						if err != nil {
							log.Errorf("Error serializing block announcement : %s", err)
							continue
						}

						log.Debugln("Publishing block announcement to the network : ", string(blockAnnouncedBytes))

						err = fullNodeAnnouncementTopic.Publish(ctx, blockAnnouncedBytes)
						if err != nil {
							log.Errorf("Error publishing block announcement to the network : %s", err)
							continue
						}
						time.Sleep(30 * time.Second)
					}
				}()

			}

		}
	}()

	// stayAliveTopic, err := ps.Join("stayAlive")
	// if err != nil {
	// 	panic(err)
	// }

	// subStayAlive, err := stayAliveTopic.Subscribe()
	// if err != nil {
	// 	panic(err)
	// }

	// // Handle incoming stay alive messages
	// go func() {
	// 	for {
	// 		msg, err := subStayAlive.Next(ctx)
	// 		if err != nil {
	// 			log.Errorln("Error getting stay alive message : ", err)
	// 		}
	// 		log.Debugln("Received stay alive message from ", msg.GetFrom().String())
	// 		uuidByte, err := uuid.FromBytes(msg.Data)
	// 		if err != nil {
	// 			log.Errorln("Error unmarshaling uuid : ", err)
	// 			continue
	// 		}
	// 		log.Debugln("Received stay alive message : ", uuidByte.String())
	// 	}
	// }()

	// // Publish stay alive messages
	// go func() {
	// 	for {
	// 		uuidByte, err := uuid.New().MarshalBinary()
	// 		if err != nil {
	// 			log.Errorln("Error marshaling uuid : ", err)
	// 			continue
	// 		}
	// 		err = stayAliveTopic.Publish(ctx, uuidByte)
	// 		if err != nil {
	// 			log.Errorln("Error publishing stay alive message : ", err)
	// 		}
	// 		log.Debugln("Published stay alive message")
	// 		// Sleep for a random duration between 1 and 5 seconds
	// 		time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
	// 	}
	// }()

	/*
	* SYNCHRONIZATION AND RE-SYNCHRONIZATION of nodes in case of absence
	 */
	// if the folder "blockchain doesn't exist, announce that the node is new and ask for the blockchain
	// if the folder has not been updated for a long time, ask for the blockchain

	var oldCid path.ImmutablePath

	go func() {
		// Add the blockchain to IPFS
		fileImmutablePathCid, err := ipfs.AddFile(ctx, nodeIpfs, ipfsApi, "./blockchain")
		if err != nil {
			log.Errorln("Error adding the blockhain to IPFS : ", err)
		}
		// Pin the file on IPFS
		pinned, err := ipfs.PinFile(ctx, ipfsApi, fileImmutablePathCid)
		if err != nil {
			log.Errorln("Error pinning the blockchain to IPFS : ", err)
		}
		log.Debugln("Blockchain pinned on IPFS : ", pinned)

		// Unpin the file on IPFS
		unpinned, err := ipfs.UnpinFile(ctx, ipfsApi, oldCid)
		if err != nil {
			log.Errorln("Error unpinning the blockchain to IPFS : ", err)
		}
		log.Debugln("Blockchain unpinned on IPFS : ", unpinned)

		oldCid = fileImmutablePathCid

		time.Sleep(300 * time.Second) // Every five minutes add and pin the last version of the blockchain on IPFS

	}()

	// Ask for the blockchain to the other full nodes

	// Join the topic FullNodeAskingForBlockchain
	fullNodeAskingForBlockchainTopic, err := ps.Join(cfg.FullNodeAskingForBlockchainStringFlag)
	if err != nil {
		panic(err)
	}
	subFullNodeAskingForBlockchain, err := fullNodeAskingForBlockchainTopic.Subscribe()
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			err = fullNodeAskingForBlockchainTopic.Publish(ctx, []byte("I need the blockchain"))
			if err != nil {
				log.Errorf("Error publishing block announcement to the network : %s", err)
				continue
			}
		}

	}()

	// Handle incoming full node asking for blockchain messages
	go func() {
		for {
			msg, err := subFullNodeAskingForBlockchain.Next(ctx)
			if err != nil {
				panic(err)
			}
			log.Debugln("Received FullNodeAskingForBlockchain message from ", msg.GetFrom().String())
			log.Debugln("Full Node Asking For Blockchain: ", string(msg.Data))
		}
	}()

	// 1. If the node is the first full node, or every node are inactive then he can create a data base
	// 2. If the node is not the first full node, he can ask to the other full nodes to send him the data base, and donwnload it and write on it after
	// 3. If the node stopping and restarting ecraer la version précédente

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
