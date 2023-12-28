// Package discovery provides functionalities to discover peers in a
// libp2p network using a Distributed Hash Table (DHT) and mDNS.
package discovery

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
)

// DHT encapsulates the functionalities of a Distributed Hash Table
// for peer discovery in a libp2p network.
type DHT struct {
	BootstrapNode             bool                  // Indicates if the node is a bootstrap node.
	RendezvousString          string                // Used for identifying peers in the network.
	BootstrapPeers            []multiaddr.Multiaddr // List of initial peers to connect to.
	DiscorveryRefreshInterval time.Duration         // Interval to refresh discovery.
	IgnoredPeers              map[peer.ID]bool      // Set of peers to ignore during discovery.
	*dht.IpfsDHT                                    // Embedded IPFS DHT instance.
}

func NewDHTDiscovery(bootstrapNode bool, rendezvousString string, bootstrapPeers []multiaddr.Multiaddr, discoveryRefreshInterval time.Duration) *DHT {
	return &DHT{
		BootstrapNode:             bootstrapNode,
		RendezvousString:          rendezvousString,
		BootstrapPeers:            bootstrapPeers,
		DiscorveryRefreshInterval: discoveryRefreshInterval,
		IgnoredPeers:              make(map[peer.ID]bool),
	}
}

// Run starts the DHT functionality of the node. It initializes the DHT, connects to
// bootstrap peers if necessary, and sets up continuous discovery of new peers.
func (d *DHT) Run(ctx context.Context, host host.Host) error {
	if err := d.startDHT(ctx, host); err != nil {
		return fmt.Errorf("starting DHT failed: %w", err)
	}

	if d.BootstrapNode {
		log.Print("BOOTSTRAP NODE - DHT IN SERVER MODE")
	} else {
		if err := d.IpfsDHT.Bootstrap(ctx); err != nil {
			return fmt.Errorf("DHT bootstrap failed: %w", err)
		}
		log.Println("Bootstrapping DHT...")
		d.bootstrapPeers(ctx, host)

		log.Println("Announcing ourselves...")
		routingDiscovery := routing.NewRoutingDiscovery(d.IpfsDHT)

		go func() {
			ticker := time.NewTicker(d.DiscorveryRefreshInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					d.announceAndConnect(ctx, host, routingDiscovery)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	return nil
}

// startDHT initializes the DHT for the given host. If the node is a bootstrap node,
// it initializes the DHT in server mode.
func (d *DHT) startDHT(ctx context.Context, host host.Host) error {
	if d.IpfsDHT != nil {
		return nil // DHT already initialized
	}

	var err error
	if d.BootstrapNode {
		d.IpfsDHT, err = dht.New(ctx, host, dht.Mode(dht.ModeServer))
	} else {
		d.IpfsDHT, err = dht.New(ctx, host)
	}
	if err != nil {
		log.Println("[startDHT] Error creating new DHT : ", err)
	}

	return err
}

// bootstrapPeers connects the host to the predefined bootstrap peers. It ensures
// the node is connected to the network and can start participating in peer discovery.
func (d *DHT) bootstrapPeers(ctx context.Context, host host.Host) {
	var wg sync.WaitGroup
	for _, peerAddr := range d.BootstrapPeers {
		peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			log.Printf("Invalid peer address: %v", err)
			continue
		}

		if host.Network().Connectedness(peerInfo.ID) == network.Connected {
			continue
		}

		wg.Add(1)
		go func(pi peer.AddrInfo) {
			defer wg.Done()
			if err := host.Connect(ctx, pi); err != nil {
				log.Println("[bootstrapPeers] Connection failed")
				// log.Printf("Connection failed to %s: %v", pi.ID, err)
			} else {
				log.Printf("Connection successful to %s", pi.ID)
			}
		}(*peerInfo)
	}
	wg.Wait()
}

// announceAndConnect advertises the node on the network and connects to discovered peers.
// It uses the provided routing discovery to find and establish connections with other peers.
func (d *DHT) announceAndConnect(ctx context.Context, host host.Host, routingDiscovery *routing.RoutingDiscovery) {
	if _, err := routingDiscovery.Advertise(ctx, d.RendezvousString); err != nil {
		log.Printf("[announceAndConnect] Error announcing self : %v", err)
		return
	}
	log.Println("[announceAndConnect] Successfully announced!")

	peersChan, err := routingDiscovery.FindPeers(ctx, d.RendezvousString)
	if err != nil {
		log.Printf("[announceAndConnect] Error finding peers : %v", err)
		return
	}

	for p := range peersChan {
		if p.ID == host.ID() || len(p.Addrs) == 0 || d.IgnoredPeers[p.ID] {
			continue
		}

		if host.Network().Connectedness(p.ID) != network.Connected {
			log.Printf("[announceAndConnect] Found peer: %s", p.ID)
			if err := host.Connect(ctx, p); err != nil {
				log.Println("[announceAndConnect] Connection failed")
				// log.Printf("[announceAndConnect] Connection failed: %v", err)
				d.IgnoredPeers[p.ID] = true
			} else {
				log.Printf("[announceAndConnect] Connection successful: %s", p.ID)
			}
		}
	}
}
