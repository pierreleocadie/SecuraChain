package discovery

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ipfs/kubo/core"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
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
	bootstrapPeers := []string{"/ip4/13.37.148.174/udp/1211/quic-v1/p2p/12D3KooWRJeqfc9RrGevpLNto8WXiYVsPhuF1qtso6dZehEY7FmP"}

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
	dhtDiscovery := NewDHTDiscovery(
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

func ConnectToBoostrapNodes(ctx context.Context, ipfsNode *core.IpfsNode) {

	// Initialize the node
	host := initializeNode(ipfsNode.PeerHost)
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
