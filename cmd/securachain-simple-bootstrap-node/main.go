package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/node"
	"github.com/pierreleocadie/SecuraChain/pkg/utils"
)

const readTimeout = 10 * time.Second

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

	/*
	* WEBPAGE
	 */
	// Web server with a page that displays the host addresses
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		addrs := host.Peerstore().Addrs(host.ID())
		for _, addr := range addrs {
			address := fmt.Sprintf("%s/p2p/%s", addr, host.ID())
			fmt.Fprintln(w, address)
		}
	})

	webserver := &http.Server{
		Addr:        ":8080",
		ReadTimeout: readTimeout,
	}

	go func() {
		err := webserver.ListenAndServe()
		if err != nil {
			log.Errorln("Error starting web server: ", err)
		}
	}()

	/*
	* PUBLISH AND SUBSCRIBE TO TOPICS
	 */
	// Create a new PubSub service using the GossipSub router
	// ps, err := pubsub.NewGossipSub(ctx, host)
	// if err != nil {
	// 	log.Errorf("Failed to create new PubSub service:", err)
	// }

	// KeepRelayConnectionAlive
	// node.PubsubKeepRelayConnectionAlive(ctx, ps, host, cfg, log)

	// NetworkVisualisation
	// networkVisualisationTopic, err := ps.Join(cfg.NetworkVisualisationStringFlag)
	// if err != nil {
	// 	log.Warnf("Failed to join NetworkVisualisation topic: %s", err)
	// }

	// Handle outgoing NetworkVisualisation messages
	// go func() {
	// 	for {
	// 		time.Sleep(5 * time.Second)
	// 		data := &visualisation.Data{
	// 			PeerID:   host.ID().String(),
	// 			NodeType: "BootstrapNode",
	// 			ConnectedPeers: func() []string {
	// 				peers := make([]string, 0)
	// 				for _, peer := range host.Network().Peers() {
	// 					// check the connectedness of the peer
	// 					if host.Network().Connectedness(peer) != network.Connected {
	// 						continue
	// 					}
	// 					peers = append(peers, peer.String())
	// 				}
	// 				return peers
	// 			}(),
	// 			TopicsList: ps.GetTopics(),
	// 			KeepRelayConnectionAlive: func() []string {
	// 				peers := make([]string, 0)
	// 				for _, peer := range ps.ListPeers("KeepRelayConnectionAlive") {
	// 					// check the connectedness of the peer
	// 					if host.Network().Connectedness(peer) != network.Connected {
	// 						continue
	// 					}
	// 					peers = append(peers, peer.String())
	// 				}
	// 				return peers
	// 			}(),
	// 			BlockAnnouncement: func() []string {
	// 				peers := make([]string, 0)
	// 				for _, peer := range ps.ListPeers("BlockAnnouncement") {
	// 					// check the connectedness of the peer
	// 					if host.Network().Connectedness(peer) != network.Connected {
	// 						continue
	// 					}
	// 					peers = append(peers, peer.String())
	// 				}
	// 				return peers
	// 			}(),
	// 			AskingBlockchain: func() []string {
	// 				peers := make([]string, 0)
	// 				for _, peer := range ps.ListPeers("AskingBlockchain") {
	// 					// check the connectedness of the peer
	// 					if host.Network().Connectedness(peer) != network.Connected {
	// 						continue
	// 					}
	// 					peers = append(peers, peer.String())
	// 				}
	// 				return peers
	// 			}(),
	// 			ReceiveBlockchain: func() []string {
	// 				peers := make([]string, 0)
	// 				for _, peer := range ps.ListPeers("ReceiveBlockchain") {
	// 					// check the connectedness of the peer
	// 					if host.Network().Connectedness(peer) != network.Connected {
	// 						continue
	// 					}
	// 					peers = append(peers, peer.String())
	// 				}
	// 				return peers
	// 			}(),
	// 			ClientAnnouncement: func() []string {
	// 				peers := make([]string, 0)
	// 				for _, peer := range ps.ListPeers("ClientAnnouncement") {
	// 					// check the connectedness of the peer
	// 					if host.Network().Connectedness(peer) != network.Connected {
	// 						continue
	// 					}
	// 					peers = append(peers, peer.String())
	// 				}
	// 				return peers
	// 			}(),
	// 			StorageNodeResponse: func() []string {
	// 				peers := make([]string, 0)
	// 				for _, peer := range ps.ListPeers("StorageNodeResponse") {
	// 					// check the connectedness of the peer
	// 					if host.Network().Connectedness(peer) != network.Connected {
	// 						continue
	// 					}
	// 					peers = append(peers, peer.String())
	// 				}
	// 				return peers
	// 			}(),
	// 			FullNodeAnnouncement: func() []string {
	// 				peers := make([]string, 0)
	// 				for _, peer := range ps.ListPeers("FullNodeAnnouncement") {
	// 					// check the connectedness of the peer
	// 					if host.Network().Connectedness(peer) != network.Connected {
	// 						continue
	// 					}
	// 					peers = append(peers, peer.String())
	// 				}
	// 				return peers
	// 			}(),
	// 			AskMyFilesList: func() []string {
	// 				peers := make([]string, 0)
	// 				for _, peer := range ps.ListPeers("AskMyFilesList") {
	// 					// check the connectedness of the peer
	// 					if host.Network().Connectedness(peer) != network.Connected {
	// 						continue
	// 					}
	// 					peers = append(peers, peer.String())
	// 				}
	// 				return peers
	// 			}(),
	// 			ReceiveMyFilesList: func() []string {
	// 				peers := make([]string, 0)
	// 				for _, peer := range ps.ListPeers("ReceiveMyFilesList") {
	// 					// check the connectedness of the peer
	// 					if host.Network().Connectedness(peer) != network.Connected {
	// 						continue
	// 					}
	// 					peers = append(peers, peer.String())
	// 				}
	// 				return peers
	// 			}(),
	// 		}
	// 		dataBytes, err := json.Marshal(data)
	// 		if err != nil {
	// 			log.Errorf("Failed to marshal NetworkVisualisation message: %s", err)
	// 			continue
	// 		}
	// 		err = networkVisualisationTopic.Publish(ctx, dataBytes)
	// 		if err != nil {
	// 			log.Errorf("Failed to publish NetworkVisualisation message: %s", err)
	// 			continue
	// 		}
	// 		log.Debugf("NetworkVisualisation message sent successfully")
	// 	}
	// }()

	/*
	* DHT DISCOVERY
	 */
	// Setup DHT discovery
	node.SetupDHTDiscovery(ctx, cfg, host, true, log)

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
