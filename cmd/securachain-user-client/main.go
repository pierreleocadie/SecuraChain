// POC node client
package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/discovery"
	ecdsaSC "github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

const (
	listeningPortFlag = 1211 // Port used for listening to incoming connections.
	RefreshInterval   = 10 * time.Second
)

var (
	ignoredPeers              map[peer.ID]bool = make(map[peer.ID]bool)
	rendezvousStringFlag                       = fmt.Sprintln("SecuraChainNetwork") // Network identifier for rendezvous string.
	channelClientAnnoncement  string           = "ClientAnnouncement"
	channelStorageNodeReponse string           = "StorageNodeResponse"
	transactionTopicNameFlag  *string          = flag.String("transactionTopicName", "NewTransaction", "ClientAnnouncement")
	ip4tcp                    string           = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 0)
	ip6tcp                    string           = fmt.Sprintf("/ip6/::/tcp/%d", 0)
	ip4quic                   string           = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", 0)
	ip6quic                   string           = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", 0)
)

func initializeNode() host.Host {
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ---------- Connection to Boostrap Nodes ----------------

	fmt.Println("\n-- Going to connect to a few boostrap nodesnodes in the Network --")

	// Initialize the node
	host := initializeNode()
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
	//--------------------

	/*
	* GENERATE ECDSA KEY PAIR FOR NODE IDENTITY
	 */
	// Generate a pair of ecdsa keys
	keyPair, err := ecdsaSC.NewECDSAKeyPair()
	if err != nil {
		log.Println("Failed to generate ecdsa key pair:", err)
	}

	/*
	* CREATE A TRNSACTION
	 */

	StringChecksum := sha256.Sum256([]byte("Y2hlY2tzdW0="))
	checksum := StringChecksum[:]

	annoncement := transaction.NewClientAnnouncement(keyPair, []byte("example_file"), []byte(".txt"), uint64(1024), checksum)

	data, err := annoncement.Serialize()
	if err != nil {
		fmt.Errorf("Error on serialize Client Annoncement transaction %s :", err)
	}

	//----------------------

	/*
	* PUBLISH AND SUBSCRIBE TO TOPICS
	 */
	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Println("Failed to create new PubSub service:", err)
	}
	// Join the topic ClientAnnoncement
	topicTrx, err := ps.Join(channelClientAnnoncement)
	if err != nil {
		log.Println("Failed to join topic:", err)
	}
	go func() {
		for {
			// Publish en byte les données
			err := topicTrx.Publish(ctx, data)
			if err != nil {
				log.Println("Failed to publish:", err)
			}
			time.Sleep(5 * time.Second)
			fmt.Println("--------------\n")
			fmt.Println(FormatClientAnnouncement(annoncement))
			fmt.Println("\n--------------")
		}
	}()

	// Join the topic StorageNodeResponse
	topicStorageNodeResponse, err := ps.Join(channelStorageNodeReponse)
	if err != nil {
		log.Println("Failed to join topic:", err)
	}

	// Subscribe to the topic StorageNodeResponse
	subStorageNodeResponse, err := topicStorageNodeResponse.Subscribe()
	if err != nil {
		log.Println("Failed to subscribe to topic:", err)
	}

	// Handle incoming chain versions in a separate goroutine
	go func() {
		for {
			msg, err := subStorageNodeResponse.Next(ctx)
			if err != nil {
				log.Println("Failed to get next chain version:", err)
			}
			fmt.Println("--------------STORAGE NODE RESPONSE-----------\n")
			log.Println(string(msg.Data))
			fmt.Println("\n--------------STORAGE NODE RESPONSE-----------")
		}
	}()

	//------------------------

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

}

func FormatClientAnnouncement(a *transaction.ClientAnnouncement) string {
	return fmt.Sprintf("AnnouncementID: %s\nOwnerAddress: %x\nFilename: %s\nExtension: %s\nFileSize: %d\nChecksum: %x\nOwnerSignature: %x\nAnnouncementTimestamp: %d",
		a.AnnouncementID, a.OwnerAddress, a.Filename, a.Extension, a.FileSize, a.Checksum, a.OwnerSignature, a.AnnouncementTimestamp)
}
