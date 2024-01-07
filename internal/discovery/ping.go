package discovery

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

func Ping(host host.Host, ctx context.Context) {
	ticker := time.NewTicker(time.Duration(rand.Intn(10)+1) * time.Second) // adjust interval to your needs
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

				log.Printf("[PING] Pinging peer %s", p)
				go func(p peer.ID) {
					// Ping the peer
					s, err := host.NewStream(ctx, p, ping.ID)
					if err != nil {
						log.Printf("Peer %s is unreachable", p)
						return
					}
					defer s.Close()
				}(p)
			}
		case <-ctx.Done():
			return
		}
	}
}
