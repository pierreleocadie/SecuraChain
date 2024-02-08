package main

import (
	"context"
	"flag"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

var yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")

func main() { //nolint: funlen
	log := ipfsLog.Logger("bootstrap-node")
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

	// Setup DHT discovery
	node.SetupDHTDiscovery(ctx, cfg, host, true)

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host, pubsub.WithMaxMessageSize(int(cfg.MaxDataRelayed)))
	if err != nil {
		log.Errorf("Failed to create GossipSub: %s", err)
	}

	// Join pubsub topics
	_, err = ps.Join(cfg.KeepRelayConnectionAliveStringFlag)
	if err != nil {
		log.Errorf("Failed to join topic: %s", err)
	}
	_, err = ps.Join(cfg.ClientAnnouncementStringFlag)
	if err != nil {
		log.Errorf("Failed to join topic: %s", err)
	}
	_, err = ps.Join(cfg.StorageNodeResponseStringFlag)
	if err != nil {
		log.Errorf("Failed to join topic: %s", err)
	}

	protocolUpdatedSub, err := host.EventBus().Subscribe(new(event.EvtPeerProtocolsUpdated))
	if err != nil {
		log.Errorf("Failed to subscribe to EvtPeerProtocolsUpdated: %s", err)
	}
	go func(sub event.Subscription) {
		for e := range sub.Out() {
			var updated bool
			for _, proto := range e.(event.EvtPeerProtocolsUpdated).Added {
				if proto == pubsub.GossipSubID_v11 || proto == pubsub.GossipSubID_v10 {
					updated = true
					break
				}
			}
			if updated {
				for _, c := range host.Network().ConnsToPeer(e.(event.EvtPeerProtocolsUpdated).Peer) {
					(*pubsub.PubSubNotif)(ps).Connected(host.Network(), c)
				}
			}
		}
	}(protocolUpdatedSub)

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
