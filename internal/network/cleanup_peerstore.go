package network

import (
	"context"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

// FIXME: This function does not work as intended. It should remove peers from the peerstore if they are unreachable.
func CleanUpPeers(log *ipfsLog.ZapEventLogger, ctx context.Context, host host.Host, cfg *config.Config) {
	ticker := time.NewTicker(cfg.DiscoveryRefreshInterval) // adjust interval to your needs
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, p := range host.Peerstore().Peers() {
				if ctx.Err() != nil { // context was canceled
					return
				}

				// Ignore our own peer ID
				if p == host.ID() {
					continue
				}

				// Ping the peer
				s, err := host.NewStream(ctx, p, ping.ID)
				if err != nil {
					// remove or mark the peer as inactive in the peerstore
					log.Errorln("Peer %s is unreachable: %v", p, err)
					host.Peerstore().RemovePeer(p)
					log.Errorln("Peer %s removed from peerstore", p)
					continue
				}
				err = s.Close()
				if err != nil {
					log.Errorln("Error closing stream: %v", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
