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
	"github.com/pierreleocadie/SecuraChain/internal/discovery"
	"github.com/pierreleocadie/SecuraChain/internal/storage/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/storage/monitoring"
	"github.com/pierreleocadie/SecuraChain/internal/storage/storageTransaction"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

const (
	listeningPortFlag = 1211 // Port used for listening to incoming connections.
)

var (
	rendezvousStringFlag = fmt.Sprintln("SecuraChainNetwork") // Network identifier for rendezvous string.
	ip4tcp               = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listeningPortFlag)
	ip6tcp               = fmt.Sprintf("/ip6/::/tcp/%d", listeningPortFlag)
	ip4quic              = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", listeningPortFlag)
	ip6quic              = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", listeningPortFlag)
	// flagExp              = flag.Bool("experimental", false, "enable experimental features")
)

func initializeNode(host host.Host) host.Host {
	/*
	* NODE INITIALIZATION
	 */
	host, err := libp2p.New(
		libp2p.UserAgent("SecuraChain"),
		libp2p.ProtocolVersion("0.0.1"),
		libp2p.ListenAddrStrings(ip4tcp, ip6tcp, ip4quic, ip6quic),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.RandomIdentity,
		libp2p.DefaultSecurity,
		libp2p.DefaultMuxers,
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
	bootstrapPeers := []string{"/ip4/13.37.148.174/udp/1211/quic-v1/p2p/12D3KooWJzCBarwtPQNbztPyHYzsh3Difo5DSAWqBWdVNuC1WA4e"}

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
		"SecuraChainNetwork",
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
	ipfsApi, nodeIpfs, err := ipfs.SpawnNode(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to spawn the node %s", err))
	}

	fmt.Println("IPFS node is running")

	// ---------- Monitoring Folder ----------------

	go func() {

		_, _, err := monitoring.WatchStorageQueueForChanges(ctx, nodeIpfs, ipfsApi)
		if err != nil {
			log.Printf("Error watchung storage queue: %s", err)
		}
	}()

	// ---------- Connection to Boostrap Nodes ----------------

	fmt.Println("\n-- Going to connect to a few boostrap nodesnodes in the Network --")

	// Initialize the node
	host := initializeNode(nodeIpfs.PeerHost)
	defer host.Close()

	// Setup DHT discovery
	setupDHTDiscovery(ctx, host)

	/*
	* DISPLAY PEER CONNECTEDNESS CHANGES
	 */
	// Subscribe to EvtPeerConnectednessChanged events
	subNet, err := host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Println("Failed to subscribe to EvtPeerConnectednessChanged: ", err)
	}
	defer subNet.Close()

	stop := make(chan bool)
	go func() {
		waitForTermSignal()
		stop <- true
	}()

	// ---------- Listennng for client's announcement ----------------

	/*
	* SUBSCRIBE TO ClientAnnouncement
	 */
	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Println("Failed to create new PubSub service:", err)
	}

	go storageTransaction.SubscribeToClientChannel(ctx, ps)
	//---------------

	// Handle connection events in a separate goroutine
	go func() {
		for {
			select {
			case <-stop:
				return // Arrête la goroutine
			case e, ok := <-subNet.Out():
				if !ok {
					return // Arrêter si le canal est femré
				}

				evt, isEventType := e.(event.EvtPeerConnectednessChanged)
				if isEventType {
					if evt.Connectedness == network.Connected {
						log.Println("Peer connected:", evt.Peer)
					} else if evt.Connectedness == network.NotConnected {
						log.Println("Peer disconnected: ", evt.Peer)
					}
				} else {
					log.Println("Received unexpected event type")
				}

			}
		}
	}()

	<-stop // Attendre que la goroutine s'arrête

	// ------------- Téléchargements du fichier --------------
	// cidDetectedFile, fileName, err := monitoring.WatchStorageQueueForChanges(ctx, nodeIpfs, ipfsApi)
	// storage.RetrieveAndSaveFileByCID(ctx, ipfsApi, cidDetectedFile)

	// ----------- Suppression du fichier -------------
	// storage.DeleteFromIPFS(ctx, ipfsApi, cidDetectedFile, fileName)

	// ------------------ HandleTransaction ------------
	// Créer un poc qui envoie une annonce dans un fichier différent

	// clientAnnoncement, err := storageTransaction.SubscribeToClientChannel(ctx, nodeIpfs)
	// if err != nil {
	// 	log.Printf("Failed on subscribing on Client Channel %s", err)
	// }
	// desaralizeAnnoncement, err := transaction.DeserializeClientAnnouncement(clientAnnoncement)
	// fmt.Println("-------------\n\n\n\n %v", desaralizeAnnoncement)

	// storageAnnoncement = storageTransaction.StorageAnnoncement(ctx, nodeIpfs, desaralizeAnnoncement, clientAnnoncement)

	fmt.Println("\nAll done!")

}
