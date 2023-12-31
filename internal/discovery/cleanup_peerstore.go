package discovery

import (
	"context"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

func CleanUpPeers(host host.Host, ctx context.Context) {
	ticker := time.NewTicker(config.PeerstoreCleanupInterval) // adjust interval to your needs
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, p := range host.Peerstore().Peers() {
				if ctx.Err() != nil { // context was canceled
					return
				}

				// Ping the peer
				s, err := host.NewStream(ctx, p, ping.ID)
				if err != nil {
					// remove or mark the peer as inactive in the peerstore
					log.Printf("Peer %s is unreachable: %v", p, err)
					host.Peerstore().RemovePeer(p)
					log.Printf("Peer %s removed from peerstore", p)
					continue
				}
				s.Close()
			}
		case <-ctx.Done():
			return
		}
	}
}
