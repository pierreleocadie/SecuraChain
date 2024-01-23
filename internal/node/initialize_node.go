package node

import (
	"context"
	"log"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/ipfs/kubo/core"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/discovery"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
)

func Initialize(cfg config.Config) host.Host {
	/*
	* NODE INITIALIZATION
	 */
	// Create a new connection manager - Exactly the same as the default connection manager but with a grace period
	connManager, err := connmgr.NewConnManager(cfg.LowWater, cfg.HighWater, connmgr.WithGracePeriod(time.Minute))
	if err != nil {
		log.Panicf("Failed to create new connection manager: %s", err)
	}

	// Create a new libp2p Host
	host, err := libp2p.New(
		libp2p.UserAgent(cfg.UserAgent),
		libp2p.ProtocolVersion(cfg.ProtocolVersion),
		libp2p.AddrsFactory(discovery.FilterOutPrivateAddrs), // Comment this line to build bootstrap node
		libp2p.EnableNATService(),
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
		libp2p.ListenAddrStrings(cfg.IP4tcp, cfg.IP6tcp, cfg.IP4quic, cfg.IP6quic),
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

func InitializeIPFSNode(ctx context.Context, log *ipfsLog.ZapEventLogger) (iface.CoreAPI, *core.IpfsNode) {
	// Spawn an IPFS node
	ipfsAPI, nodeIpfs, err := ipfs.SpawnNode(ctx)
	if err != nil {
		log.Panicf("Failed to spawn IPFS node: %s", err)
	}

	log.Debugf("IPFS node spawned with PeerID: %s", nodeIpfs.Identity.String())

	return ipfsAPI, nodeIpfs
}
