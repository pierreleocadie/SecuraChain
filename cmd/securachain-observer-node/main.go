package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
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

var (
	yamlConfigFilePath = flag.String("config", "", "Path to the yaml config file")
	clients            = make(map[*websocket.Conn]bool) // Enregistrer les clients connectés
	clientsMutex       sync.Mutex
	upgrader           = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Ne faites pas cela dans une application de production
		},
	}
	data    []visualisation.Data
	oldData []visualisation.Data
	mapData = make(map[string]visualisation.Data)
)

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
	// node.SetupDHTDiscovery(ctx, cfg, host, false, log)

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

	// KeepRelayConnectionAlive
	node.PubsubKeepRelayConnectionAlive(ctx, ps, host, cfg, log)

	// NetworkVisualisation
	networkVisualisationTopic, err := ps.Join(cfg.NetworkVisualisationStringFlag)
	if err != nil {
		log.Warnf("Failed to join NetworkVisualisation topic: %s", err)
	}

	subNetworkVisualisation, err := networkVisualisationTopic.Subscribe()
	if err != nil {
		log.Warnf("Failed to subscribe to NetworkVisualisation topic: %s", err)
	}

	// Handle incoming NetworkVisualisation messages
	go func() {
		for {
			msg, err := subNetworkVisualisation.Next(ctx)
			if err != nil {
				log.Warnf("Failed to get next message from NetworkVisualisation topic: %s", err)
			}
			peerData := visualisation.Data{}
			err = json.Unmarshal(msg.Data, &peerData)
			if err != nil {
				log.Warnf("Failed to unmarshal NetworkVisualisation message: %s", err)
			}
			log.Debugln("Received NetworkVisualisation message from: ", msg.GetFrom())
			mapData[peerData.PeerID] = peerData
			if len(oldData) == 0 && len(data) == 0 {
				data = append(data, peerData)
			} else {
				newData := []visualisation.Data{}
				oldData = append(oldData, data...)
				// Delete all peers that have an empty ConnectedPeers slice in the map, that means they are disconnected
				for key, value := range mapData {
					if len(value.ConnectedPeers) == 0 {
						delete(mapData, key)
					}
				}
				// Transform the map into a slice
				for _, value := range mapData {
					newData = append(newData, value)
				}
				// Delete data
				data = nil
				// Add the new data
				data = append(data, newData...)
			}
		}
	}()

	/*
	* NETWORK VISUALISATION - WEBSOCKET SERVER
	 */
	http.HandleFunc("/ws", visualisation.CreateHandler(upgrader, data, &clientsMutex, clients, log))
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Infoln("ListenAndServe: ", err)
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
