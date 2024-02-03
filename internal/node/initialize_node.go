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
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
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
	var h host.Host

	// Create a new connection manager - Exactly the same as the default connection manager but with a grace period
	connManager, err := connmgr.NewConnManager(cfg.LowWater, cfg.HighWater, connmgr.WithGracePeriod(time.Minute))
	if err != nil {
		log.Panicf("Failed to create new connection manager: %s", err)
	}

	hostReady := make(chan struct{})
	defer close(hostReady)

	hostGetter := func() host.Host {
		<-hostReady // closed when we finish setting up the host
		return h
	}

	// Create a new libp2p Host
	h, err = libp2p.New(
		libp2p.UserAgent(cfg.UserAgent),
		libp2p.ProtocolVersion(cfg.ProtocolVersion),
		libp2p.AddrsFactory(discovery.FilterOutPrivateAddrs), // Comment this line to build bootstrap node
		libp2p.EnableNATService(),
		// libp2p.NATPortMap(),
		// libp2p.EnableHolePunching(),
		libp2p.ListenAddrStrings(cfg.IP4tcp, cfg.IP6tcp, cfg.IP4quic, cfg.IP6quic),
		libp2p.ConnectionManager(connManager),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.RandomIdentity,
		libp2p.DefaultSecurity,
		libp2p.DefaultMuxers,
		libp2p.DefaultEnableRelay,
		libp2p.EnableRelayService(),
		libp2p.EnableAutoRelayWithPeerSource(
			func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
				r := make(chan peer.AddrInfo)
				defer close(r)
				host := hostGetter()
				if host == nil { // context canceled etc.
					return r
				}
				for _, p := range host.Network().Peers() {
					peerProtocols, err := host.Peerstore().GetProtocols(p)
					if err != nil {
						// log.Errorln("Error getting peer protocols : ", err)
						continue
					}
					for _, protocol := range peerProtocols {
						if protocol == "/libp2p/circuit/relay/0.2.0/hop" || protocol == "/libp2p/circuit/relay/0.2.0/stop" {
							select {
							case r <- host.Peerstore().PeerInfo(p):
							case <-ctx.Done():
								return r
							default:
								return r
							}
						}
					}
				}
				return r
			},
			autorelay.WithBackoff(10*time.Second),
			autorelay.WithBootDelay(time.Minute),
			autorelay.WithMinInterval(10*time.Second),
		),
	)
	if err != nil {
		log.Panicf("Failed to create new libp2p Host: %s", err)
	}
	log.Printf("Our node ID: %s\n", h.ID())

	// Node info
	hostInfo := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}

	addrs, err := peer.AddrInfoToP2pAddrs(&hostInfo)
	if err != nil {
		log.Panicf("Failed to convert peer.AddrInfo to p2p.Addr: %s", err)
	}

	for _, addr := range addrs {
		log.Println("Node address: ", addr)
	}

	for _, addr := range h.Addrs() {
		log.Println("Listening on address: ", addr)
	}

	return h
}

func InitializeIPFSNode(ctx context.Context, cfg *config.Config, log *ipfsLog.ZapEventLogger) (iface.CoreAPI, *core.IpfsNode) {
	// Spawn an IPFS node
	ipfsAPI, nodeIpfs, err := ipfs.SpawnNode(ctx, cfg)
	if err != nil {
		log.Panicf("Failed to spawn IPFS node: %s", err)
	}

	log.Debugf("IPFS node spawned with PeerID: %s", nodeIpfs.Identity.String())

	return ipfsAPI, nodeIpfs
}
