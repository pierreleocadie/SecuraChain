package main

import (
	"context"
	"encoding/json"
	"flag"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	netwrk "github.com/pierreleocadie/SecuraChain/internal/network"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/internal/visualisation"
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
	_ = node.SetupDHTDiscovery(ctx, cfg, h, false)

	/*
	 * NETWORK PEER DISCOVERY WITH mDNS
	 */
	// Initialize mDNS
	mdnsConfig := netwrk.NewMDNSDiscovery(cfg.RendezvousStringFlag)
	// Run MDNS
	if err := mdnsConfig.Run(h); err != nil {
		log.Fatalf("Failed to run mDNS: %s", err)
		return
	}

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Panicf("Failed to create GossipSub: %s", err)
	}

	// VisualisationData
	visualisationDataTopic, err := ps.Join("visualisationData")
	if err != nil {
		log.Warnf("Failed to join VisualisationData topic: %s", err)
	}

	// Handle outgoing VisualisationData messages
	go func() {
		for {
			time.Sleep(5 * time.Second)
			connectedPeers := make([]string, 0, len(h.Network().Peers()))
			for _, peer := range h.Network().Peers() {
				connectedPeers = append(connectedPeers, peer.String())
			}

			visualisationData := visualisation.VisualisationData{
				Sender:                   h.ID().String(),
				ConnectedPeers:           connectedPeers,
				TopicsList:               ps.GetTopics(),
				ClientAnnouncement:       []string{},
				StorageNodeResponse:      []string{},
				KeepRelayConnectionAlive: []string{},
			}

			visualisationDataJSON, err := json.Marshal(visualisationData)
			if err != nil {
				log.Errorf("Failed to marshal visualisationData: %s", err)
				continue
			}

			err = visualisationDataTopic.Publish(ctx, visualisationDataJSON)
			if err != nil {
				log.Errorf("Failed to publish visualisationData message: %s", err)
				continue
			}
			log.Debugf("visualisationData message sent successfully")
		}
	}()

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
