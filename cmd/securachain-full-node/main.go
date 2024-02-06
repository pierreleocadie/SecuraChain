package main

import (
	"context"
	"flag"
	"fmt"
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
		log.Panicf("Failed to join block announcement topic : %s", err)
	}
	subBlockAnnouncement, err := blockAnnouncementTopic.Subscribe()
	if err != nil {
		log.Panicf("Failed to subscribe to block announcement topic : %s", err)
	}

	// Join the topic FullNodeAnnouncementStringFlag
	fullNodeAnnouncementTopic, err := ps.Join(cfg.FullNodeAnnouncementStringFlag)
	if err != nil {
		panic(err)
	}
	// subFullNodeAnnouncement, err := fullNodeAnnouncementTopic.Subscribe()
	// if err != nil {
	// 	panic(err)
	// }

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

			// blockPrevBlock, err := block.DeserializeBlock(blockAnnounced.PrevBlock)
			// if err != nil {
			// 	log.Errorf("Error deserializing block announcement : %s", err)
			// 	continue
			// }

			/*
			* Validate blocks coming from minors
			* Manage random selection if several blocks arrive at the same time
			* Add the block to the blockchain
			* Add the blockchain to IPFS
			* Publish the block to the network
			 */

			blockBuff := make(map[int64][]*block.Block)
			listOfBlocks, err := fullnode.HandleIncomingBlock(blockAnnounced, blockBuff, databaseInstance)

			blockAnnouncedBytes, err := blocks[0].Serialize()
			if err != nil {
				fmt.Printf("Errior serializing block announcement: %s\n", err)
				return nil
			}
			return

			if len(listOfBlocks) > 1 {
				// Return all blocks with the same timestamp for the minor node to select based on the longest chain
				continue
			}

			var oldCid path.ImmutablePath

			fullnode.AddBlockchainToIPFS(ctx, nodeIpfs, ipfsApi, oldCid)

			// Publish the block to the network
			fmt.Println("Publishing block announcement to the network : ", string(blockBytes))

			err = fullNodeAnnouncementTopic.Publish(ctx, blockBytes)
			if err != nil {
				fmt.Println("Error publishing block announcement to the network : ", err)

			}

		}
	}()

	// If I go the blockchain, and still up to date
	// If I also one time pushed the blockchain to the network with IPFS
	// Handle incoming full node announcement messages
	go func() {

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
