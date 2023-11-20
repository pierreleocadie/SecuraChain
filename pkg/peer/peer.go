package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	peerstore "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/multiformats/go-multiaddr"
)

func main() {
	node, err := createLibp2pNode()
	if err != nil {
		panic(err)
	}
	defer shutdownNode(node)

	pingService := configurePingService(node)

	printNodeInfo(node)

	handleCommandLineArgs(node, pingService)
}

func createLibp2pNode() (host.Host, error) {
	return libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Ping(false),
	)
}

func configurePingService(node host.Host) *ping.PingService {
	pingService := &ping.PingService{Host: node}
	node.SetStreamHandler(ping.ID, pingService.PingHandler)
	return pingService
}

func printNodeInfo(node host.Host) {
	fullAddr := getFullAddress(node)
	fmt.Println("Node address:", fullAddr)
}

func getFullAddress(node host.Host) string {
	peerInfo := peerstore.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}
	addrs, err := peerstore.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		panic(err)
	}

	// Append the transport address to the multiaddr
	return addrs[0].String()
}

func handleCommandLineArgs(node host.Host, pingService *ping.PingService) {
	if len(os.Args) > 1 {
		remotePeerAddr := os.Args[1]
		connectToPeer(node, pingService, remotePeerAddr)
	} else {
		waitForSignal()
	}
}

func connectToPeer(node host.Host, pingService *ping.PingService, remotePeerAddr string) {
	if len(os.Args) > 1 {
		addr, err := multiaddr.NewMultiaddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		peer, err := peerstore.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		if err := node.Connect(context.Background(), *peer); err != nil {
			panic(err)
		}
		fmt.Println("sending 5 ping messages to", addr)
		ch := pingService.Ping(context.Background(), peer.ID)
		for i := 0; i < 5; i++ {
			res := <-ch
			fmt.Println("pinged", addr, "in", res.RTT)
		}
	}
}

func waitForSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Received signal, shutting down...")
}

func shutdownNode(node host.Host) {
	if err := node.Close(); err != nil {
		panic(err)
	}
}
