package main

import (
	"context"
	"flag"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
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
	ipfsLog.SetLogLevel("user-client", "DEBUG")

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
	_, nodeIpfs, err := ipfs.SpawnNode(ctx)
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
	defer pebbleDB.Close()

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

			// Deserialize the block announcement
			blockAnnounced, err := block.DeserializeBlock(msg.Data)
			if err != nil {
				log.Errorf("Error deserializing block announcement : %s", err)
				continue
			}

			/*
			* BLOCK VALIDATION
			 */

			// Deserialize the previous block
			prevBlock, err := block.DeserializeBlock(blockAnnounced.PrevBlock)
			if err != nil {
				log.Errorf("Error deserializing previous block : %s", err)
				continue
			}

			// Validate the block
			if !consensus.ValidateBlock(blockAnnounced, prevBlock) {
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

					}
				}()

			}

		}
	}()

	stayAliveTopic, err := ps.Join("stayAlive")
	if err != nil {
		panic(err)
	}

	subStayAlive, err := stayAliveTopic.Subscribe()
	if err != nil {
		panic(err)
	}

	// Handle incoming stay alive messages
	go func() {
		for {
			msg, err := subStayAlive.Next(ctx)
			if err != nil {
				log.Errorln("Error getting stay alive message : ", err)
			}
			log.Debugln("Received stay alive message from ", msg.GetFrom().String())
			uuidByte, err := uuid.FromBytes(msg.Data)
			if err != nil {
				log.Errorln("Error unmarshaling uuid : ", err)
				continue
			}
			log.Debugln("Received stay alive message : ", uuidByte.String())
		}
	}()

	// Publish stay alive messages
	go func() {
		for {
			uuidByte, err := uuid.New().MarshalBinary()
			if err != nil {
				log.Errorln("Error marshaling uuid : ", err)
				continue
			}
			err = stayAliveTopic.Publish(ctx, uuidByte)
			if err != nil {
				log.Errorln("Error publishing stay alive message : ", err)
			}
			log.Debugln("Published stay alive message")
			// Sleep for a random duration between 1 and 5 seconds
			time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
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
