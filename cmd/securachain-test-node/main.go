package main

import (
	"context"
	"flag"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
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
	var h host.Host

	// Create a new connection manager - Exactly the same as the default connection manager but with a grace period
	connManager, err := connmgr.NewConnManager(cfg.LowWater, cfg.HighWater, connmgr.WithGracePeriod(time.Minute))
	if err != nil {
		log.Panicf("Failed to create new connection manager: %s", err)
	}

	hostReady := make(chan struct{})
	hostGetter := func() host.Host {
		<-hostReady // closed when we finish setting up the host
		return h
	}

	// Create a new libp2p Host
	h, err = libp2p.New(
		libp2p.UserAgent(cfg.UserAgent),
		libp2p.ProtocolVersion(cfg.ProtocolVersion),
		libp2p.AddrsFactory(netwrk.FilterOutPrivateAddrs), // Comment this line to build bootstrap node
		libp2p.EnableNATService(),
		// libp2p.NATPortMap(),
		// libp2p.EnableHolePunching(),
		libp2p.ListenAddrStrings(cfg.IP4tcp, cfg.IP6tcp, cfg.IP4quic, cfg.IP6quic),
		libp2p.ConnectionManager(connManager),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.RandomIdentity,
		libp2p.DefaultSecurity,
		libp2p.DefaultMuxers,
		libp2p.DefaultEnableRelay,
		libp2p.EnableAutoRelayWithPeerSource(
			netwrk.NewPeerSource(log, hostGetter),
			autorelay.WithBackoff(cfg.DiscoveryRefreshInterval),
			autorelay.WithMinInterval(cfg.DiscoveryRefreshInterval),
			autorelay.WithNumRelays(1),
			autorelay.WithMinCandidates(1),
		),
	)
	if err != nil {
		log.Panicf("Failed to create new libp2p Host: %s", err)
	}
	log.Debugf("Our node ID: %s\n", h.ID())

	// Close the hostReady channel to signal that the host is ready
	close(hostReady)

	// Node info
	hostInfo := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}

	addrs, err := peer.AddrInfoToP2pAddrs(&hostInfo)
	if err != nil {
		log.Panicf("Failed to convert peer.AddrInfo to p2p.Addr: %s", err)
	}

	for _, addr := range addrs {
		log.Debugln("Node address: ", addr)
	}

	for _, addr := range h.Addrs() {
		log.Debugln("Listening on address: ", addr)
	}
	defer h.Close()

	/*
	* PUBSUB
	 */
	// gossipSubRt := pubsub.DefaultGossipSubRouter(host)
	// ps, err := pubsub.NewGossipSubWithRouter(ctx, host, gossipSubRt)
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Panicf("Failed to create GossipSub: %s", err)
	}

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

	/*
	* DHT DISCOVERY
	 */
	// Setup DHT discovery
	_ = node.SetupDHTDiscovery(ctx, cfg, h, false)

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
