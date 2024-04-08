// Package discovery provides functionality for peer discovery in a libp2p network.
package network

import (
	"context"
	"fmt"
	"log"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// MDNS struct encapsulates the parameters for mDNS-based peer discovery.
type MDNS struct {
	Rendezvous string // Rendezvous string used to identify peers in the mDNS service.
}

// discoveryNotifee is an implementation of the mdns.Notifee interface.
// It is used to handle notifications about peers discovered via mDNS.
type discoveryNotifee struct {
	host host.Host // The libp2p host used to connect to discovered peers.
}

// NewMDNSDiscovery creates a new MDNS discovery instance with the specified rendezvous string.
func NewMDNSDiscovery(rendezvous string) *MDNS {
	return &MDNS{
		Rendezvous: rendezvous,
	}
}

// HandlePeerFound is called when a new peer is discovered via mDNS.
// It attempts to connect the local host to the discovered peer.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if err := n.host.Connect(context.Background(), pi); err != nil {
		log.Printf("mDNS: error connecting to peer %s: %v", pi.ID, err)
	}
}

// Run starts the mDNS discovery service with the configured parameters.
// It initializes and starts an mDNS service for peer discovery within the local network.
func (m *MDNS) Run(log *ipfsLog.ZapEventLogger, host host.Host) error {
	n := &discoveryNotifee{host}

	// Create a new mDNS service using the provided host and rendezvous string.
	mdnsService := mdns.NewMdnsService(host, m.Rendezvous, n)
	if err := mdnsService.Start(); err != nil {
		log.Errorln("mDNS: error starting service")
		return fmt.Errorf("mDNS: error starting service: %w", err)
	}

	return nil
}
