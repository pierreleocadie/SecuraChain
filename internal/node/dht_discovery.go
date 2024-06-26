package node

import (
	"context"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/network"
)

func SetupDHTDiscovery(ctx context.Context, cfg *config.Config, host host.Host, bootstrapNode bool, log *ipfsLog.ZapEventLogger) *network.DHT {
	/*
	* NETWORK PEER DISCOVERY WITH DHT
	 */
	// Convert the bootstrap peers from string to multiaddr
	var bootstrapPeersMultiaddr []multiaddr.Multiaddr
	if !bootstrapNode {
		for _, peer := range cfg.BootstrapPeers {
			peerMultiaddr, err := multiaddr.NewMultiaddr(peer)
			if err != nil {
				log.Errorln("Error converting bootstrap peer to multiaddr : ", err)
				return nil
			}
			bootstrapPeersMultiaddr = append(bootstrapPeersMultiaddr, peerMultiaddr)
		}
	}

	// Initialize DHT in server mode
	dhtDiscovery := network.NewDHTDiscovery(
		bootstrapNode,
		cfg.RendezvousStringFlag,
		bootstrapPeersMultiaddr,
		cfg.DiscoveryRefreshInterval,
	)

	// Run DHT
	if err := dhtDiscovery.Run(ctx, host, log); err != nil {
		log.Fatalf("Failed to run DHT: %s", err)
		return nil
	}

	return dhtDiscovery
}
