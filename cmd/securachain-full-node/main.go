package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/discovery"
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

	log.Debugf("Full node initialized with PeerID: %s", host.ID().String())

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

	/*
	* Ask for the blockchain to the other full nodes if the node don't meet the requirements
	 */

	if fullnode.NeedToFetchBlockchain() {
		log.Debugln("Blockchain doesn't exist or is not up to date")
		databaseInstance = fullnode.FetchBlockchain(ctx, 2*time.Minute, ps)
	} else {
		log.Debugln("Blockchain exists and is up to date")
	}

	/*
	* Wait for blocks coming from minors
	 */

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
	// subFullNodeAnnouncement, err := fullNodeAnnouncementTopic.Subscribe()
	// if err != nil {
	// 	panic(err)
	// }

	// Join the topic MinorConflicts
	MinorConflictsTopic, err := ps.Join("MinorConflicts")
	if err != nil {
		log.Panicf("Failed to join minor conflicts topic : %s\n", err)
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
				log.Debugln("Error deserializing block announcement : %s", err)
				continue
			}

			/*
			* Validate blocks coming from minors
			* Manage conflicts
			* Add the block to the blockchain
			* Publish the block to the network
			* Add the blockchain to IPFS
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

			// Serialize the block
			blockBytes, err := listOfBlocks[0].Serialize()
			if err != nil {
				log.Debugf("error serializing block : %s\n", err)
				continue
			}

			// Publish the block to the network (for minors and indexing and searching nodes)
			log.Debugln("Publishing block announcement to the network :", string(blockBytes))

			err = fullNodeAnnouncementTopic.Publish(ctx, blockBytes)
			if err != nil {
				log.Debugln("Error publishing block announcement to the network : ", err)

			}

			// Send the blockchain to IPFS
			cidBlockChain, err = fullnode.AddBlockchainToIPFS(ctx, nodeIpfs, ipfsApi, cidBlockChain)
			if err != nil {
				log.Debugf("error adding the blockchain to IPFS : %s\n", err)
				continue
			}

		}
	}()

	// Handle incoming asking blockchain messages from full nodes

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
