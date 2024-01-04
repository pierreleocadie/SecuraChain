package discovery

import (
	"context"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

func Ping(host host.Host, ctx context.Context) {
	ticker := time.NewTicker(config.PeerstoreCleanupInterval) // adjust interval to your needs
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, p := range host.Network().Peers() {
				if ctx.Err() != nil { // context was canceled
					return
				}

				// Ignore our own peer ID
				if p == host.ID() {
					continue
				}

				go func(p peer.ID) {
					// Ping the peer
					s, err := host.NewStream(ctx, p, ping.ID)
					if err != nil {
						log.Printf("Peer %s is unreachable: %v", p, err)
					}
					s.Close()
				}(p)
			}
		case <-ctx.Done():
			return
		}
	}
}
