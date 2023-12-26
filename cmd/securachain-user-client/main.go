// POC node client
package main

import (
	"context"
	"crypto/sha256"
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
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	"github.com/pierreleocadie/SecuraChain/internal/discovery"
	ecdsaSC "github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

const (
	listeningPortFlag = 0 // Port used for listening to incoming connections.
	RefreshInterval   = 10 * time.Second
	lowWater          = 160
	highWater         = 192
)

var (
	rendezvousStringFlag = fmt.Sprintln("SecuraChainNetwork")
	ip4tcp               = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listeningPortFlag)
	ip6tcp               = fmt.Sprintf("/ip6/::/tcp/%d", listeningPortFlag)
	ip4quic              = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", listeningPortFlag)
	ip6quic              = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", listeningPortFlag)
	bootstrapPeers       = []string{
		"/ip4/13.37.148.174/udp/1211/quic-v1/p2p/12D3KooWBm6aEtcGiJNsnsCwaiH4SoqJHZMgvctdQsyAenwyt8Ds",
	}
	clientAnnouncementStringFlag  = fmt.Sprintln("ClientAnnouncement")
	storageNodeResponseStringFlag = fmt.Sprintln("StorageNodeResponse")
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
	// Convert the bootstrap peers from string to multiaddr
	var bootstrapPeersMultiaddr []multiaddr.Multiaddr
	for _, peer := range bootstrapPeers {
		peerMultiaddr, err := multiaddr.NewMultiaddr(peer)
		if err != nil {
			log.Println("Error converting bootstrap peer to multiaddr : ", err)
			return
		}
		bootstrapPeersMultiaddr = append(bootstrapPeersMultiaddr, peerMultiaddr)
	}

	// Initialize DHT in server mode
	dhtDiscovery := discovery.NewDHTDiscovery(
		false,
		rendezvousStringFlag,
		bootstrapPeersMultiaddr,
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
	size := 1024
	annoncement := transaction.NewClientAnnouncement(keyPair, []byte("example_file"), []byte(".txt"), uint64(size), checksum)

	data, err := annoncement.Serialize()
	if err != nil {
		log.Printf("Error on serialize Client Annoncement transaction %s :", err)
	}

	/*
	* PUBLISH AND SUBSCRIBE TO TOPICS
	 */
	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Println("Failed to create new PubSub service:", err)
	}
	// Join the topic ClientAnnoncement
	topicClient, err := ps.Join(clientAnnouncementStringFlag)
	if err != nil {
		log.Println("Failed to join topic:", err)
	}

	go func() {
		for {
			// Publish en byte les données
			err := topicClient.Publish(ctx, data)
			if err != nil {
				log.Println("Failed to publish:", err)
			}
			time.Sleep(5 * time.Second)
			fmt.Println("-------------- CLIENT ANNOUNCEMENT -------------")
			fmt.Println(FormatClientAnnouncement(annoncement))
			fmt.Println("\n-------------- CLIENT ANNOUNCEMENT -------------")
		}
	}()

	// Join the topic StorageNodeResponse
	topicStorageNodeResponse, err := ps.Join(storageNodeResponseStringFlag)
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
			fmt.Println("--------------STORAGE NODE RESPONSE-----------")
			log.Println(string(msg.Data))
			fmt.Println("\n--------------STORAGE NODE RESPONSE-----------")
		}
	}()

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
	return fmt.Sprintf(
		"AnnouncementID: %s\nOwnerAddress: %x\nFilename: %s\nExtension: %s\nFileSize: %d\nChecksum: %x\nOwnerSignature: %x\nAnnouncementTimestamp: %d",
		a.AnnouncementID, a.OwnerAddress, a.Filename, a.Extension, a.FileSize, a.Checksum, a.OwnerSignature, a.AnnouncementTimestamp)
}
