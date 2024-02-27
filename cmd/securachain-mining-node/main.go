package main

import (
	"context"
	"flag"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

var (
	yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")
	generateKeys       = flag.Bool("genKeys", false, "Generate new ECDSA and AES keys to the paths specified in the config file")
)

func main() { //nolint: funlen
	log := ipfsLog.Logger("mining-node")
	err := ipfsLog.SetLogLevel("mining-node", "DEBUG")
	if err != nil {
		log.Errorln("Error setting log level : ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flag.Parse()

	// Load the config file
	cfg := node.LoadConfig(yamlConfigFilePath, log)

	if *generateKeys {
		node.GenerateKeys(cfg, log)
	}

	ecdsaKeyPair, _ := node.LoadKeys(cfg, log)

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
	node.SetupDHTDiscovery(ctx, cfg, host, false)

	/*
	* PUBSUB
	 */
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Panicf("Failed to create new pubsub: %s", err)
	}

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
