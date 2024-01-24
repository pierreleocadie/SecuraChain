// Poc Minor with bloc creaiton and mining
package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
)

var yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")

func main() {

	log := ipfsLog.Logger("storage-node")
	ipfsLog.SetLogLevel("storage-node", "DEBUG")

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

	// /*
	// * IPFS NODE
	//  */
	// // Spawn an IPFS node
	// _, nodeIpfs, err := ipfs.SpawnNode(ctx)
	// if err != nil {
	// 	log.Fatalf("Failed to spawn IPFS node: %s", err)
	// }

	// log.Debugf("IPFS node spawned with PeerID: %s", nodeIpfs.Identity.String())

	/*
	* NODE LIBP2P
	 */
	// Initialize the storage node
	host := node.Initialize(*cfg)
	defer host.Close()

	// Setup DHT discovery
	node.SetupDHTDiscovery(ctx, host, false)

	log.Debugf("Storage node initialized with PeerID: %s", host.ID().String())

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// Join the BlockAnnouncement topic
	blockAnnouncementTopic, err := ps.Join(cfg.BlockAnnouncementStringFlag)
	if err != nil {
		panic(err)
	}

	// // Subscribe to the BlockAnnouncement topic
	// subBlockAnnouncement, err := blockAnnouncementTopic.Subscribe()
	// if err != nil {
	// 	panic(err)

	// }

	// Create a sample block
	minerKeyPair, _ := ecdsa.NewECDSAKeyPair()
	transactions := []transaction.Transaction{}

	newBlock := block.NewBlock(transactions, []byte("GenesisBlock"), 1, minerKeyPair)

	consensus.MineBlock(newBlock)

	// Handle publishing of blocks in a separate goroutine
	go func() {
		for {
			// Publish the block announcement
			newBlockBytes, err := newBlock.Serialize()
			if err != nil {
				log.Errorln("Failed to serialize block:", err)
			}

			log.Debugln("Publishing block : ", newBlockBytes)
			err = blockAnnouncementTopic.Publish(ctx, newBlockBytes)
			if err != nil {
				log.Errorln("Failed to publish block:", err)
				continue
			}
			time.Sleep(15 * time.Second)

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
