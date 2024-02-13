package main

import (
	"context"
	"flag"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
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
	h := node.Initialize(log, *cfg)
	defer h.Close()
	log.Debugf("Storage node initialized with PeerID: %s", h.ID().String())

	/*
	* RELAY SERVICE
	 */
	// Check if the node is behind NAT
	behindNAT := netwrk.NATDiscovery(log)

	// If the node is behind NAT, search for a node that supports relay
	// TODO: Optimize this code
	if !behindNAT {
		log.Debugln("Node is not behind NAT")
		// Start the relay service
		_, err = relay.New(h,
			relay.WithResources(relay.Resources{
				Limit: &relay.RelayLimit{
					// Duration is the time limit before resetting a relayed connection; defaults to 2min.
					Duration: cfg.MaxRelayedConnectionDuration,
					// Data is the limit of data relayed (on each direction) before resetting the connection.
					// Defaults to 128KB
					// Data: 1 << 30, // 1 GB
					Data: cfg.MaxDataRelayed,
				},
				// Default values
				// ReservationTTL is the duration of a new (or refreshed reservation).
				// Defaults to 1hr.
				ReservationTTL: cfg.RelayReservationTTL,
				// MaxReservations is the maximum number of active relay slots; defaults to 128.
				MaxReservations: cfg.MaxRelayReservations,
				// MaxCircuits is the maximum number of open relay connections for each peer; defaults to 16.
				MaxCircuits: cfg.MaxRelayCircuits,
				// BufferSize is the size of the relayed connection buffers; defaults to 2048.
				BufferSize: cfg.MaxRelayedConnectionBufferSize,

				// MaxReservationsPerPeer is the maximum number of reservations originating from the same
				// peer; default is 4.
				MaxReservationsPerPeer: cfg.MaxRelayReservationsPerPeer,
				// MaxReservationsPerIP is the maximum number of reservations originating from the same
				// IP address; default is 8.
				MaxReservationsPerIP: cfg.MaxRelayReservationsPerIP,
				// MaxReservationsPerASN is the maximum number of reservations origination from the same
				// ASN; default is 32
				MaxReservationsPerASN: cfg.MaxRelayReservationsPerASN,
			}),
		)
		if err != nil {
			log.Errorln("Error instantiating relay service : ", err)
		}
		log.Debugln("Relay service started")
	} else {
		log.Debugln("Node is behind NAT")
	}

	/*
	* DHT DISCOVERY
	 */
	// Setup DHT discovery
	_ = node.SetupDHTDiscovery(ctx, cfg, h, false)

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Panicf("Failed to create GossipSub: %s", err)
	}

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
			if msg.GetFrom().String() == h.ID().String() {
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
			// 	log.Warnf("Not enough peers in KeepRelayConnectionAlive topic")
			// 	continue
			// }
			err := keepRelayConnectionAliveTopic.Publish(ctx, netwrk.GeneratePacket(h.ID()))
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
