package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"reflect"
	"sync"
	"time"

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
	data          []visualisation.Data = []visualisation.Data{}
	oldData       []visualisation.Data = []visualisation.Data{}
	mapData                            = make(map[string]visualisation.Data)
	mapDataMutex  sync.Mutex
	peersDataChan = make(chan visualisation.Data, 200)
)

func main() { //nolint: funlen
	log := ipfsLog.Logger("observer-node")
	err := ipfsLog.SetLogLevel("observer-node", "INFO")
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

	// Handle outgoing NetworkVisualisation messages
	go func() {
		for {
			time.Sleep(5 * time.Second)
			data := &visualisation.Data{
				PeerID:   host.ID().String(),
				NodeType: "FullNode",
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
			peersDataChan <- peerData
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Second)
			// Delete all peers that have an empty ConnectedPeers slice in the map, that means they are disconnected
			mapDataMutex.Lock()
			for key, value := range mapData {
				if len(value.ConnectedPeers) == 0 {
					delete(mapData, key)
				}
			}
			mapDataMutex.Unlock()
		}
	}()

	go func() {
		i := 0
		for range time.Tick(time.Second) {
			peerData := <-peersDataChan
			mapDataMutex.Lock()
			if _, exists := mapData[peerData.PeerID]; exists {
				// Check if the data is different
				if reflect.DeepEqual(mapData[peerData.PeerID], peerData) {
					mapDataMutex.Unlock()
					continue
				}
			} else {
				i++
				log.Infoln("New peer connected: ", peerData.PeerID, " - ", i)
			}
			//log.Infoln("Received NetworkVisualisation message from: ", msg.GetFrom())
			mapDataMutex.Lock()
			mapData[peerData.PeerID] = peerData
			mapDataMutex.Unlock()
			if len(oldData) == 0 && len(data) == 0 {
				data = append(data, peerData)
				visualisation.SendDataToClients(&clientsMutex, clients, visualisation.WebSocketMessage{Type: "initData", Data: data}, log)
			} else {
				oldData = append(oldData, data...)
				mapDataMutex.Lock()
				newData := make([]visualisation.Data, 0, len(mapData))
				// Transform the map into a slice
				for _, value := range mapData {
					newData = append(newData, value)
				}
				mapDataMutex.Unlock()
				// Delete data
				data = nil
				// Add the new data
				data = append(data, newData...)
				diffData := visualisation.NewDiffData(oldData, data)
				visualisation.SendDataToClients(&clientsMutex, clients, visualisation.WebSocketMessage{Type: "dataUpdate", Data: diffData}, log)
			}
			mapDataMutex.Unlock()
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
