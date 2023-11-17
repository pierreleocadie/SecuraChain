package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"

	"github.com/pierreleocadie/SecuraChain/internal/discovery"
)

var (
	logMessages          [][]byte
	logMutex             sync.Mutex
	rendezvousStringFlag *string = flag.String("rendezvousString", "SecuraChainNetwork", "Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	listeningPortFlag    *int    = flag.Int("listeningPort", 1211, "Port on which to listen for new connections. Use 0 to listen on a random port")
	ip4tcp               string  = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *listeningPortFlag)
	ip6tcp               string  = fmt.Sprintf("/ip6/::/tcp/%d", *listeningPortFlag)
	ip4quic              string  = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", *listeningPortFlag)
	ip6quic              string  = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", *listeningPortFlag)
)

func handleWebPage(w http.ResponseWriter, r *http.Request) {
	for _, message := range logMessages {
		fmt.Fprintf(w, "%s\n", string(message))
	}
}

func addLogMessage(message []byte) {
	logMutex.Lock()
	defer logMutex.Unlock()
	logMessages = append(logMessages, message)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	/*
	* WEB SERVER - USE ONLY FOR DEVELOPMENT PURPOSES
	 */
	// This web server is used to display the network analytics in a web browser
	// It is only used for development purposes
	// It is not necessary for the node to work
	// The web server is started on port 1212
	// Start the web server
	http.HandleFunc("/", handleWebPage)
	go func() {
		if err := http.ListenAndServe(":1212", nil); err != nil {
			log.Fatal("Failed to start web server: ", err)
		}
	}()
	log.Println("Web server started on http://localhost:1212")

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

	/*
	* NETWORK PEER DISCOVERY WITH DHT
	 */
	// Initialize DHT in server mode
	dhtDiscovery := discovery.NewDHTDiscovery(
		true,
		*rendezvousStringFlag,
		[]multiaddr.Multiaddr{},
		0,
	)

	// Run DHT
	if err := dhtDiscovery.Run(ctx, host); err != nil {
		log.Fatal("Failed to run DHT: ", err)
	}

	/*
	* PUBLISH AND SUBSCRIBE TO TOPICS
	 */
	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Println("Failed to create new PubSub service:", err)
	}

	// Subscribe to the SecuraChainNetworkAnalytics topic
	// This topic will be used by other nodes to send analytics data while the development of the network
	// We don't need to publish anything to this topic, just subscribe to it to receive analytics data
	networkAnalyticsTopic, err := ps.Join("SecuraChainNetworkAnalytics")
	if err != nil {
		log.Fatal("Failed to join SecuraChainNetworkAnalytics topic:", err)
	}

	// Subscribe to the SecuraChainNetwork topic
	subNetworkAnalytics, err := networkAnalyticsTopic.Subscribe()
	if err != nil {
		log.Fatal("Failed to subscribe to SecuraChainNetworkAnalytics topic:", err)
	}

	// Every 2 hours, we clear the logMessages slice
	// This is only used for development purposes
	// We don't want to keep all the analytics data in memory
	go func() {
		for {
			time.Sleep(2 * time.Hour)
			logMutex.Lock()
			logMessages = nil
			logMutex.Unlock()
		}
	}()

	// Handle messages in a separate goroutine
	go func() {
		for {
			msg, err := subNetworkAnalytics.Next(ctx)
			if err != nil {
				log.Printf("Failed to get next message from SecuraChainNetworkAnalytics topic: %v", err)
				continue
			}
			// Add the log message to the web server page
			// This is only used for development purposes
			// Append message and date to logMessages slice -> date format: 2006-01-02 15:04:05
			addLogMessage([]byte(fmt.Sprintf("%s - %s", time.Now().Format("2006-01-02 15:04:05"), string(msg.Data))))
		}
	}()

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

	// wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Received signal, shutting down...")
	if err := host.Close(); err != nil {
		panic(err)
	}
}
