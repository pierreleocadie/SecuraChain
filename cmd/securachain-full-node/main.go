package main

import (
	"context"
	"flag"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pierreleocadie/SecuraChain/internal/config"
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
