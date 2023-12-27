package node

import (
	"context"
	"log"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/discovery"
)

func SetupDHTDiscovery(ctx context.Context, host host.Host, bootstrapNode bool) {
	/*
	* NETWORK PEER DISCOVERY WITH DHT
	 */
	// Convert the bootstrap peers from string to multiaddr
	var bootstrapPeersMultiaddr []multiaddr.Multiaddr
	if !bootstrapNode {
		for _, peer := range config.BootstrapPeers {
			peerMultiaddr, err := multiaddr.NewMultiaddr(peer)
			if err != nil {
				log.Println("Error converting bootstrap peer to multiaddr : ", err)
				return
			}
			bootstrapPeersMultiaddr = append(bootstrapPeersMultiaddr, peerMultiaddr)
		}
	}

	// Initialize DHT in server mode
	dhtDiscovery := discovery.NewDHTDiscovery(
		bootstrapNode,
		config.RendezvousStringFlag,
		bootstrapPeersMultiaddr,
		config.DHTDiscoveryRefreshInterval,
	)

	// Run DHT
	if err := dhtDiscovery.Run(ctx, host); err != nil {
		log.Println("Failed to run DHT: ", err)
		return
	}
}
