package node

import (
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

func Initialize() host.Host {
	/*
	* NODE INITIALIZATION
	 */
	// Create a new connection manager - Exactly the same as the default connection manager but with a grace period
	connManager, err := connmgr.NewConnManager(config.LowWater, config.HighWater, connmgr.WithGracePeriod(time.Minute))
	if err != nil {
		log.Panicf("Failed to create new connection manager: %s", err)
	}

	// Create a new libp2p Host
	host, err := libp2p.New(
		libp2p.UserAgent(config.UserAgent),
		libp2p.ProtocolVersion(config.ProtocolVersion),
		libp2p.EnableNATService(),
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
		libp2p.ListenAddrStrings(config.Ip4tcp, config.Ip6tcp, config.Ip4quic, config.Ip6quic),
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
		log.Panicf("Failed to create new libp2p Host: %s", err)
	}
	log.Printf("Our node ID: %s\n", host.ID())

	// Node info
	hostInfo := peer.AddrInfo{
		ID:    host.ID(),
		Addrs: host.Addrs(),
	}

	addrs, err := peer.AddrInfoToP2pAddrs(&hostInfo)
	if err != nil {
		log.Panicf("Failed to convert peer.AddrInfo to p2p.Addr: %s", err)
	}

	for _, addr := range addrs {
		log.Println("Node address: ", addr)
	}

	for _, addr := range host.Addrs() {
		log.Println("Listening on address: ", addr)
	}

	return host
}
