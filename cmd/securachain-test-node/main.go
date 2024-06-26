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
	log := ipfsLog.Logger("test-node")
	err := ipfsLog.SetLogLevel("*", "DEBUG")
	if err != nil {
		log.Errorln("Error setting log level : ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flag.Parse()

	// Load the config file
	if *yamlConfigFilePath == "" {
		log.Panicln("Please provide a path to the yaml config file")
	}

	cfg, err := config.LoadConfig(*yamlConfigFilePath)
	if err != nil {
		log.Panicln("Error loading config file : ", err)
	}

	/*
	* NODE LIBP2P
	 */
	h := node.Initialize(log, *cfg)
	defer h.Close()
	log.Debugf("Storage node initialized with PeerID: %s", h.ID().String())

	/*
	* DHT DISCOVERY
	 */
	// Setup DHT discovery
	_ = node.SetupDHTDiscovery(ctx, cfg, h, false, log)

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Panicf("Failed to create GossipSub: %s", err)
	}

	// KeepRelayConnectionAlive
	keepRelayConnectionAliveTopic, err := ps.Join(cfg.KeepRelayConnectionAliveStringFlag)
	if err != nil {
		log.Warnf("Failed to join KeepRelayConnectionAlive topic: %s", err)
	}

	// Subscribe to KeepRelayConnectionAlive topic
	subKeepRelayConnectionAlive, err := keepRelayConnectionAliveTopic.Subscribe()
	if err != nil {
		log.Warnf("Failed to subscribe to KeepRelayConnectionAlive topic: %s", err)
	}

	pubsubHub := &node.PubSubHub{
		ClientAnnouncementTopic:       nil,
		ClientAnnouncementSub:         nil,
		StorageNodeResponseTopic:      nil,
		StorageNodeResponseSub:        nil,
		KeepRelayConnectionAliveTopic: keepRelayConnectionAliveTopic,
		KeepRelayConnectionAliveSub:   subKeepRelayConnectionAlive,
		BlockAnnouncementTopic:        nil,
		BlockAnnouncementSub:          nil,
		AskingBlockchainTopic:         nil,
		AskingBlockchainSub:           nil,
		ReceiveBlockchainTopic:        nil,
		ReceiveBlockchainSub:          nil,
		AskMyFilesTopic:               nil,
		AskMyFilesSub:                 nil,
		SendMyFilesTopic:              nil,
		SendMyFilesSub:                nil,
		NetworkVisualisationTopic:     nil,
		NetworkVisualisationSub:       nil,
	}

	// KeepRelayConnectionAlive
	node.PubsubKeepRelayConnectionAlive(ctx, pubsubHub, h, cfg, log)

	/*
	* DISPLAY PEER CONNECTEDNESS CHANGES
	 */
	// Subscribe to EvtPeerConnectednessChanged events
	subNet, err := h.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Errorln("Failed to subscribe to EvtPeerConnectednessChanged: ", err)
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

	utils.WaitForTermSignal()
}
