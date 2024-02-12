package node

import (
	"context"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/ipfs/kubo/core"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/ipfs"
	"github.com/pierreleocadie/SecuraChain/internal/network"
)

func Initialize(log *ipfsLog.ZapEventLogger, cfg config.Config) host.Host { //nolint: funlen
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
	hostGetter := func() host.Host {
		<-hostReady // closed when we finish setting up the host
		return h
	}

	// Create a new libp2p Host
	h, err = libp2p.New(
		libp2p.UserAgent(cfg.UserAgent),
		libp2p.ProtocolVersion(cfg.ProtocolVersion),
		libp2p.AddrsFactory(network.FilterOutPrivateAddrs), // Comment this line to build bootstrap node
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
		libp2p.EnableAutoRelayWithPeerSource(
			network.NewPeerSource(log, hostGetter),
			autorelay.WithBackoff(cfg.DiscoveryRefreshInterval),
			autorelay.WithMinInterval(cfg.DiscoveryRefreshInterval),
			autorelay.WithNumRelays(1),
			autorelay.WithMinCandidates(1),
		),
	)
	if err != nil {
		log.Panicf("Failed to create new libp2p Host: %s", err)
	}
	log.Debugf("Our node ID: %s\n", h.ID())

	// Close the hostReady channel to signal that the host is ready
	close(hostReady)

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
		log.Debugln("Node address: ", addr)
	}

	for _, addr := range h.Addrs() {
		log.Debugln("Listening on address: ", addr)
	}

	/*
	* RELAY SERVICE
	 */
	// Check if the node is behind NAT
	behindNAT := network.NATDiscovery(log)

	// If the node is behind NAT, search for a node that supports relay
	// TODO: Optimize this code
	if !behindNAT {
		log.Debugln("Node is not behind NAT")
		// Start the relay service
		_, err = relay.New(h,
			relay.WithResources(relay.Resources{
				Limit: &relay.RelayLimit{
					// Duration is the time limit before resetting a relayed connection; defaults to 2min.
					Duration: cfg.MaxRelayedConnectionDuration,
					// Data is the limit of data relayed (on each direction) before resetting the connection.
					// Defaults to 128KB
					// Data: 1 << 30, // 1 GB
					Data: cfg.MaxDataRelayed,
				},
				// Default values
				// ReservationTTL is the duration of a new (or refreshed reservation).
				// Defaults to 1hr.
				ReservationTTL: cfg.RelayReservationTTL,
				// MaxReservations is the maximum number of active relay slots; defaults to 128.
				MaxReservations: cfg.MaxRelayReservations,
				// MaxCircuits is the maximum number of open relay connections for each peer; defaults to 16.
				MaxCircuits: cfg.MaxRelayCircuits,
				// BufferSize is the size of the relayed connection buffers; defaults to 2048.
				BufferSize: cfg.MaxRelayedConnectionBufferSize,

				// MaxReservationsPerPeer is the maximum number of reservations originating from the same
				// peer; default is 4.
				MaxReservationsPerPeer: cfg.MaxRelayReservationsPerPeer,
				// MaxReservationsPerIP is the maximum number of reservations originating from the same
				// IP address; default is 8.
				MaxReservationsPerIP: cfg.MaxRelayReservationsPerIP,
				// MaxReservationsPerASN is the maximum number of reservations origination from the same
				// ASN; default is 32
				MaxReservationsPerASN: cfg.MaxRelayReservationsPerASN,
			}),
		)
		if err != nil {
			log.Errorln("Error instantiating relay service : ", err)
		}
		log.Debugln("Relay service started")
	} else {
		log.Debugln("Node is behind NAT")
	}

	log.Infof("Host protocols are: %v", h.Mux().Protocols())

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
