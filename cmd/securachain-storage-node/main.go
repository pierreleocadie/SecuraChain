package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/discovery"
	"github.com/pierreleocadie/SecuraChain/internal/storage/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/storage/monitoring"
	storagetransaction "github.com/pierreleocadie/SecuraChain/internal/storage/storage_transaction"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

const (
	listeningPortFlag = 0 // Port used for listening to incoming connections.
	lowWater          = 160
	highWater         = 192
)

var (
	rendezvousStringFlag         = fmt.Sprintln("SecuraChainNetwork") // Network identifier for rendezvous string.
	ip4tcp                       = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listeningPortFlag)
	ip6tcp                       = fmt.Sprintf("/ip6/::/tcp/%d", listeningPortFlag)
	ip4quic                      = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", listeningPortFlag)
	ip6quic                      = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", listeningPortFlag)
	clientAnnouncementStringFlag = fmt.Sprintln("ClientAnnouncement")
	// storageNodeResponseStringFlag = fmt.Sprintln("StorageNodeResponse")
	// flagExp              = flag.Bool("experimental", false, "enable experimental features")
)

func initializeNode() host.Host {
	/*
	* NODE INITIALIZATION
	 */
	// Create a new connection manager - Exactly the same as the default connection manager but with a grace period
	connManager, err := connmgr.NewConnManager(lowWater, highWater, connmgr.WithGracePeriod(time.Minute))
	if err != nil {
		panic(err)
	}

	// Create a new libp2p Host
	host, err := libp2p.New(
		libp2p.UserAgent("SecuraChain"),
		libp2p.ProtocolVersion("0.0.1"),
		libp2p.EnableNATService(),
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
		libp2p.ListenAddrStrings(ip4tcp, ip6tcp, ip4quic, ip6quic),
		libp2p.ConnectionManager(connManager),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.RandomIdentity,
		libp2p.DefaultSecurity,
		libp2p.DefaultMuxers,
		libp2p.DefaultEnableRelay,
		libp2p.EnableRelayService(),
	)
	if err != nil {
		panic(err)
	}
	log.Printf("Our node ID: %s\n", host.ID())

	// Node info
	hostInfo := peer.AddrInfo{
		ID:    host.ID(),
		Addrs: host.Addrs(),
	}

	addrs, err := peer.AddrInfoToP2pAddrs(&hostInfo)
	if err != nil {
		panic(err)
	}

	for _, addr := range addrs {
		log.Println("Node address: ", addr)
	}

	for _, addr := range host.Addrs() {
		log.Println("Listening on address: ", addr)
	}

	return host
}

func setupDHTDiscovery(ctx context.Context, host host.Host) {
	/*
	* NETWORK PEER DISCOVERY WITH DHT
	 */
	// TODO : remove hard coded boostrap node
	bootstrapPeers := []string{"/ip4/13.37.148.174/udp/1211/quic-v1/p2p/12D3KooWBm6aEtcGiJNsnsCwaiH4SoqJHZMgvctdQsyAenwyt8Ds"}

	bootstrapPeersAddrs := make([]multiaddr.Multiaddr, len(bootstrapPeers))
	for i, peerAddr := range bootstrapPeers {
		peerAddrMA, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			log.Println("Failed to parse multiaddress: ", err)
			return
		}
		bootstrapPeersAddrs[i] = peerAddrMA
	}
	// Initialize DHT in client mode
	dhtDiscovery := discovery.NewDHTDiscovery(
		false,
		rendezvousStringFlag,
		bootstrapPeersAddrs,
		10*time.Second,
	)

	// Run DHT
	if err := dhtDiscovery.Run(ctx, host); err != nil {
		log.Println("Failed to run DHT: ", err)
		return
	}
}
func waitForTermSignal() {
	// wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Received signal, shutting down...")
}

func main() {
	// ---------- Part I: Getting a IPFS node running  -------------------

	fmt.Println("-- Getting an IPFS node running -- ")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Créer un noeud de stockage ipfs sur la machine
	ipfsAPI, nodeIpfs, err := ipfs.SpawnNode(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to spawn the node %s", err))
	}

	fmt.Println("IPFS node is running")

	// ---------- Monitoring Folder ----------------

	go func() {
		_, _, err := monitoring.WatchStorageQueueForChanges(ctx, nodeIpfs, ipfsAPI)
		if err != nil {
			log.Printf("Error watchung storage queue: %s", err)
		}
	}()

	// ---------- Connection to Boostrap Nodes ----------------

	fmt.Println("\n-- Going to connect to a few boostrap nodesnodes in the Network --")

	// Initialize the node
	host := initializeNode()
	defer host.Close()

	// Setup DHT discovery
	setupDHTDiscovery(ctx, host)

	// ---------- Listenning for client's announcement ----------------

	/*
	* SUBSCRIBE TO ClientAnnouncement
	 */
	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Println("Failed to create new PubSub service:", err)
	}

	announceChan := make(chan *transaction.ClientAnnouncement)
	// go storageTransaction.SubscribeToClientChannel(ctx, ps, announceChan)

	// // Join the topic client channel
	// topicClient, err := ps.Join(channelClientAnnoncement)
	// if err != nil {
	// 	log.Println("Failed to join topic:", err)
	// }
	// subClient, err := topicClient.Subscribe()
	// if err != nil {
	// 	log.Println("Failed to subscribe to topic client:", err)
	// }

	// go func() {
	// 	for {
	// 		msg, err := subClient.Next(ctx)
	// 		if err != nil {
	// 			log.Println("Failed to read next message:", err)
	// 			continue
	// 		}
	// 		fmt.Printf("Received message: %s\n", string(msg.Data))
	// 		// transac, err := transaction.DeserializeClientAnnouncement(msg.Data)
	// 		// if err != nil {
	// 		// 	log.Printf("Error on desarializing Client announcement %s", err)
	// 		// 	continue
	// 		// }
	// 		// announceChan <- transac // Envoyez l'annonce sur le canal
	// 	}
	// }()

	//---------------------------------------------------------------
	// Join the topic clientAnnouncementStringFlag
	clientAnnouncementTopic, err := ps.Join(clientAnnouncementStringFlag)
	if err != nil {
		panic(err)
	}

	// Subscribe to clientAnnouncementStringFlag topic
	subClientAnnouncement, err := clientAnnouncementTopic.Subscribe()
	if err != nil {
		panic(err)
	}

	// Handle incoming ClientAnnouncement messages
	go func() {
		for {
			msg, err := subClientAnnouncement.Next(ctx)
			if err != nil {
				panic(err)
			}
			log.Println("Received ClientAnnouncement message from ", msg.GetFrom().String())
			log.Println(string(msg.Data))
			transac, err := transaction.DeserializeClientAnnouncement(msg.Data)
			if err != nil {
				log.Printf("Error on desarializing Client announcement %s", err)
				continue
			}
			announceChan <- transac
		}
	}()

	// ---------- Sending StorageNodeResponse ----------------

	go storagetransaction.StorageAnnoncement(ctx, ps, nodeIpfs, <-announceChan)

	//--------------------------------------------------------------

	/*
	* DISPLAY PEER CONNECTEDNESS CHANGES
	 */
	// Subscribe to EvtPeerConnectednessChanged events
	subNet, err := host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Println("Failed to subscribe to EvtPeerConnectednessChanged: ", err)
	}
	defer subNet.Close()

	// Handle connection events in a separate goroutine
	go func() {
		for e := range subNet.Out() {
			evt := e.(event.EvtPeerConnectednessChanged)
			if evt.Connectedness == network.Connected {
				log.Println("Peer connected:", evt.Peer)
			} else if evt.Connectedness == network.NotConnected {
				log.Println("Peer disconnected: ", evt.Peer)
			}
		}
	}()

	waitForTermSignal()

	// ------------- Téléchargements du fichier --------------
	// cidDetectedFile, fileName, err := monitoring.WatchStorageQueueForChanges(ctx, nodeIpfs, ipfsAPI)
	// storage.RetrieveAndSaveFileByCID(ctx, ipfsAPI, cidDetectedFile)

	// ----------- Suppression du fichier -------------
	// storage.DeleteFromIPFS(ctx, ipfsAPI, cidDetectedFile, fileName)

	fmt.Println("\nAll done!")
}
