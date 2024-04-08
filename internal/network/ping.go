package network

import (
	"context"
	"crypto/rand"
	"math/big"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

func Ping(log *ipfsLog.ZapEventLogger, ctx context.Context, host host.Host, randomInterval int64) {
	// Generate a cryptographically secure random number between 1 and 10
	n, err := rand.Int(rand.Reader, big.NewInt(randomInterval))
	if err != nil {
		log.Errorln("Failed to generate secure random number: %v", err)
	}

	ticker := time.NewTicker(time.Duration(n.Int64()+1) * time.Second) // adjust interval to your needs
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

				log.Debugln("[PING] Pinging peer %s", p)
				go func(p peer.ID) {
					// Ping the peer
					s, err := host.NewStream(ctx, p, ping.ID)
					if err != nil {
						log.Errorln("Peer %s is unreachable", p)
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
