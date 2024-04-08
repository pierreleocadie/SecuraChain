package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
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
	data          []visualisation.Data = []visualisation.Data{}
	oldData       []visualisation.Data = []visualisation.Data{}
	mapData                            = make(map[string]visualisation.Data)
	peersDataChan                      = make(chan []byte, 200)
)

func main() { //nolint: funlen
	log := ipfsLog.Logger("observer-node")
	err := ipfsLog.SetLogLevel("observer-node", "DEBUG")
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

	// pubsub.GossipSubD = 8
	// pubsub.GossipSubDlo = 5
	// pubsub.GossipSubDhi = 10
	// pubsub.GossipSubDscore = 8
	// pubsub.GossipSubDout = 8
	// pubsub.GossipSubDlazy = 8
	// pubsub.GossipSubGossipRetransmission = 12

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
	node.SetupDHTDiscovery(ctx, cfg, host, false, log)

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
	ps, err := pubsub.NewGossipSub(ctx, host,
		pubsub.WithValidateQueueSize(1000),
		pubsub.WithPeerOutboundQueueSize(1000),
		pubsub.WithValidateWorkers(runtime.NumCPU()*2),
		pubsub.WithValidateThrottle(8192*2),
	)
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

	subNetworkVisualisation, err := networkVisualisationTopic.Subscribe(pubsub.WithBufferSize(1000))
	if err != nil {
		log.Warnf("Failed to subscribe to NetworkVisualisation topic: %s", err)
	}

	// Before starting we need to add each bootstrap peers data to the mapData
	// and data slice to avoid problems with the visualisation of the network
	// if we have an only node in the network connected to all the bootstrap peers
	// we will have some problems to get data from the bootstrap peers
	// so we need to add the bootstrap peers data to the mapData and data slice
	// to avoid this problem
	for _, p := range cfg.BootstrapPeers {
		// Convert the bootstrap peer from string to multiaddr
		peerMultiaddr, err := multiaddr.NewMultiaddr(p)
		if err != nil {
			log.Errorln("Error converting bootstrap peer to multiaddr : ", err)
			return
		}
		// Get the peer ID from the multiaddr
		peerIDstr, err := peerMultiaddr.ValueForProtocol(multiaddr.P_P2P)
		if err != nil {
			log.Errorln("Error getting peer ID from multiaddr : ", err)
			return
		}
		// Convert the peer ID from string to peer.ID
		peerID, err := peer.Decode(peerIDstr)
		if err != nil {
			log.Errorln("Error decoding peer ID : ", err)
			return
		}
		// Check the connectedness of the peer
		if host.Network().Connectedness(peerID) != network.Connected {
			continue
		}
		// Get the peer data
		peerData := visualisation.Data{
			PeerID:                   peerID.String(),
			NodeType:                 "BootstrapNode",
			ConnectedPeers:           []string{host.ID().String()},
			TopicsList:               []string{},
			KeepRelayConnectionAlive: []string{},
			BlockAnnouncement:        []string{},
			AskingBlockchain:         []string{},
			ReceiveBlockchain:        []string{},
			ClientAnnouncement:       []string{},
			StorageNodeResponse:      []string{},
			FullNodeAnnouncement:     []string{},
			AskMyFilesList:           []string{},
			ReceiveMyFilesList:       []string{},
		}
		// Add the peer data to the mapData
		mapData[peerData.PeerID] = peerData
		// Add the peer data to the data slice
		data = append(data, peerData)
	}

	log.Infoln("Bootstrap peers data added to the mapData and data slice")
	log.Infof("Number of bootstrap peers data added: %d", len(cfg.BootstrapPeers))
	log.Infof("Number of bootstrap peers data in the mapData: %d", len(mapData))
	log.Infof("Data slice : %v", data)

	// Handle outgoing NetworkVisualisation messages
	go func() {
		for {
			time.Sleep(5 * time.Second)
			data := &visualisation.Data{
				PeerID:   host.ID().String(),
				NodeType: "ObserverNode",
				ConnectedPeers: func() []string {
					peers := make([]string, 0)
					for _, peer := range host.Network().Peers() {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				TopicsList: ps.GetTopics(),
				KeepRelayConnectionAlive: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("KeepRelayConnectionAlive") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				BlockAnnouncement: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("BlockAnnouncement") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				AskingBlockchain: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("AskingBlockchain") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				ReceiveBlockchain: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("ReceiveBlockchain") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				ClientAnnouncement: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("ClientAnnouncement") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				StorageNodeResponse: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("StorageNodeResponse") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				FullNodeAnnouncement: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("FullNodeAnnouncement") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				AskMyFilesList: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("AskMyFilesList") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
				ReceiveMyFilesList: func() []string {
					peers := make([]string, 0)
					for _, peer := range ps.ListPeers("ReceiveMyFilesList") {
						// check the connectedness of the peer
						if host.Network().Connectedness(peer) != network.Connected {
							continue
						}
						peers = append(peers, peer.String())
					}
					return peers
				}(),
			}
			dataBytes, err := json.Marshal(data)
			if err != nil {
				log.Errorf("Failed to marshal NetworkVisualisation message: %s", err)
				continue
			}
			err = networkVisualisationTopic.Publish(ctx, dataBytes)
			if err != nil {
				log.Errorf("Failed to publish NetworkVisualisation message: %s", err)
				continue
			}
			log.Debugf("NetworkVisualisation message sent successfully")
		}
	}()

	// Handle incoming NetworkVisualisation messages
	go func() {
		peersReceived := make(map[string]bool)
		for {
			msg, err := subNetworkVisualisation.Next(ctx)
			if err != nil {
				log.Warnf("Failed to get next message from NetworkVisualisation topic: %s", err)
			}
			peersReceived[msg.GetFrom().String()] = true
			log.Debug("Number of peers received: ", len(peersReceived))
			peersDataChan <- msg.Data
		}
	}()

	go func() {
		i := 0
		for {
			select {
			case peerDataByte := <-peersDataChan:
				peerData := visualisation.Data{}
				err = json.Unmarshal(peerDataByte, &peerData)
				if err != nil {
					log.Warnf("Failed to unmarshal NetworkVisualisation message: %s", err)
				}
				_, exists := mapData[peerData.PeerID]
				if exists && reflect.DeepEqual(mapData[peerData.PeerID], peerData) {
					continue
				}
				if !exists {
					i++
					log.Infoln("New peer added to mapData: ", i)
				}
				//log.Infoln("Received NetworkVisualisation message from: ", msg.GetFrom())
				mapData[peerData.PeerID] = peerData
				if len(oldData) == 0 && len(data) == 0 {
					data = append(data, peerData)
					visualisation.SendDataToClients(&clientsMutex, clients, visualisation.WebSocketMessage{Type: "initData", Data: data}, log)
				} else {
					oldData = append(oldData, data...)
					newData := make([]visualisation.Data, 0, len(mapData))
					// Transform the map into a slice
					for _, value := range mapData {
						newData = append(newData, value)
					}
					for key, value := range mapData {
						if len(value.ConnectedPeers) == 0 {
							delete(mapData, key)
						}
					}
					// Delete data
					data = nil
					// Add the new data
					data = append(data, newData...)
					diffData := visualisation.NewDiffData(oldData, data)
					visualisation.SendDataToClients(&clientsMutex, clients, visualisation.WebSocketMessage{Type: "dataUpdate", Data: diffData}, log)
				}
			}
		}
	}()

	/*
	* NETWORK VISUALISATION - WEBSOCKET SERVER
	 */
	http.HandleFunc("/ws", visualisation.CreateHandler(upgrader, &data, &clientsMutex, clients, log))
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
