package main

import (
	"context"
	"flag"
	"time"

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

func main() { //nolint: funlen, gocyclo
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
	// Initialize the node
	host := node.Initialize(log, *cfg)
	defer host.Close()

	/*
	* PUBSUB
	 */
	// gossipSubRt := pubsub.DefaultGossipSubRouter(host)
	// ps, err := pubsub.NewGossipSubWithRouter(ctx, host, gossipSubRt)
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Panicf("Failed to create GossipSub: %s", err)
	}

	/*
	* DHT DISCOVERY
	 */
	// Setup DHT discovery
	_ = node.SetupDHTDiscovery(ctx, cfg, host, false)

	go func() {
		// Print protocols supported by peers we are connected to
		for {
			time.Sleep(5 * time.Second)
			peers := host.Network().Peers()
			for _, p := range peers {
				protocols, err := host.Peerstore().GetProtocols(p)
				if err != nil {
					log.Errorf("Failed to get protocols for peer %s: %s", p.String(), err)
					continue
				}
				log.Infof("Peer %s supports protocols: %v", p.String(), protocols)
			}
		}
	}()

	// protocolUpdatedSub, err := host.EventBus().Subscribe(new(event.EvtPeerProtocolsUpdated))
	// if err != nil {
	// 	log.Errorf("Failed to subscribe to EvtPeerProtocolsUpdated: %s", err)
	// }
	// go func(sub event.Subscription) {
	// 	for {
	// 		select {
	// 		case e, ok := <-sub.Out():
	// 			if !ok {
	// 				return
	// 			}
	// 			var updated bool
	// 			for _, proto := range e.(event.EvtPeerProtocolsUpdated).Added {
	// 				if proto == pubsub.GossipSubID_v11 || proto == pubsub.GossipSubID_v10 {
	// 					updated = true
	// 					break
	// 				}
	// 			}
	// 			if updated {
	// 				for _, c := range host.Network().ConnsToPeer(e.(event.EvtPeerProtocolsUpdated).Peer) {
	// 					(*pubsub.PubSubNotif)(ps).Connected(host.Network(), c)
	// 				}
	// 			}
	// 		}
	// 	}
	// }(protocolUpdatedSub)

	// KeepRelayConnectionAlive
	keepRelayConnectionAliveTopic, err := ps.Join("KeepRelayConnectionAlive")
	if err != nil {
		log.Warnf("Failed to join KeepRelayConnectionAlive topic: %s", err)
	}

	// Subscribe to KeepRelayConnectionAlive topic
	subKeepRelayConnectionAlive, err := keepRelayConnectionAliveTopic.Subscribe()
	if err != nil {
		log.Warnf("Failed to subscribe to KeepRelayConnectionAlive topic: %s", err)
	}

	// Handle incoming KeepRelayConnectionAlive messages
	go func() {
		for {
			msg, err := subKeepRelayConnectionAlive.Next(ctx)
			if err != nil {
				log.Errorf("Failed to get next message from KeepRelayConnectionAlive topic: %s", err)
				continue
			}
			if msg.GetFrom().String() == host.ID().String() {
				continue
			}
			log.Debugf("Received KeepRelayConnectionAlive message from %s", msg.GetFrom().String())
			log.Debugf("KeepRelayConnectionAlive: %s", string(msg.Data))
		}
	}()

	// Handle outgoing KeepRelayConnectionAlive messages
	go func() {
		for {
			time.Sleep(cfg.KeepRelayConnectionAliveInterval)
			// if !gossipSubRt.EnoughPeers("KeepRelayConnectionAlive", 1) {
			// 	continue
			// }
			err := keepRelayConnectionAliveTopic.Publish(ctx, netwrk.GeneratePacket(host.ID()))
			if err != nil {
				log.Errorf("Failed to publish KeepRelayConnectionAlive message: %s", err)
				continue
			}
			log.Debugf("KeepRelayConnectionAlive message sent successfully")
		}
	}()

	go func() {
		for {
			time.Sleep(5 * time.Second)
			peers := ps.ListPeers("KeepRelayConnectionAlive")
			log.Debugf("Peers in KeepRelayConnectionAlive topic: %v", peers)
		}
	}()

	/*
	* DISPLAY PEER CONNECTEDNESS CHANGES
	 */
	// Subscribe to EvtPeerConnectednessChanged events
	subNet, err := host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
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
