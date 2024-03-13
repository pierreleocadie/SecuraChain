package main

import (
	"context"
	"flag"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	netwrk "github.com/pierreleocadie/SecuraChain/internal/network"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

var yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")

func main() { //nolint: funlen
	log := ipfsLog.Logger("observer-node")
	err := ipfsLog.SetLogLevel("*", "DEBUG")
	if err != nil {
		log.Errorln("Error setting log level : ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flag.Parse()

	// Load the config file
	if *yamlConfigFilePath == "" {
		log.Panicln("Please provide a path to the yaml config file. Flag: -config <path/to/config.yaml>")
	}

	cfg, err := config.LoadConfig(*yamlConfigFilePath)
	if err != nil {
		log.Panicln("Error loading config file : ", err)
	}

	/*
	* NODE LIBP2P
	 */
	// Initialize the node
	host := node.Initialize(log, *cfg)
	defer host.Close()

	/*
	* DHT DISCOVERY
	 */
	// Setup DHT discovery
	node.SetupDHTDiscovery(ctx, cfg, host, true)

	/*
	 * NETWORK PEER DISCOVERY WITH mDNS
	 */
	// Initialize mDNS
	mdnsConfig := netwrk.NewMDNSDiscovery(cfg.RendezvousStringFlag)
	// Run MDNS
	if err := mdnsConfig.Run(host); err != nil {
		log.Fatalf("Failed to run mDNS: %s", err)
		return
	}

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Panicf("Failed to create GossipSub: %s", err)
	}

	// VisualisationData
	visualisationDataTopic, err := ps.Join("visualisationData")
	if err != nil {
		log.Warnf("Failed to join VisualisationData topic: %s", err)
	}

	// Subscribe to VisualisationData topic
	subVisualisationData, err := visualisationDataTopic.Subscribe()
	if err != nil {
		log.Warnf("Failed to subscribe to VisualisationData topic: %s", err)
	}

	// Handle incoming VisualisationData messages
	go func() {
		for {
			msg, err := subVisualisationData.Next(ctx)
			if err != nil {
				log.Errorf("Failed to get next message from VisualisationData topic: %s", err)
				continue
			}
			log.Debugf("Received VisualisationData message from %s", msg.GetFrom().String())
			log.Debugf("VisualisationData: %s", string(msg.Data))
		}
	}()

	/*
	* DISPLAY PEER CONNECTEDNESS CHANGES
	 */
	// Subscribe to EvtPeerConnectednessChanged events
	subNet, err := host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Debugln("Failed to subscribe to EvtPeerConnectednessChanged: ", err)
	}
	defer subNet.Close()

	// Handle connection events in a separate goroutine
	go func() {
		for e := range subNet.Out() {
			evt := e.(event.EvtPeerConnectednessChanged)
			if evt.Connectedness == network.Connected {
				log.Debugln("Peer connected:", evt.Peer)
			} else if evt.Connectedness == network.NotConnected {
				log.Debugln("Peer disconnected: ", evt.Peer)
			}
		}
	}()

	// Wait for a termination signal
	utils.WaitForTermSignal()
}
