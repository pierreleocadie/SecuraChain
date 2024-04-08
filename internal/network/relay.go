package network

import (
	"context"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
)

func NewPeerSource(log *ipfsLog.ZapEventLogger, hostGetter func() host.Host) autorelay.PeerSource {
	return func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
		r := make(chan peer.AddrInfo, numPeers)
		defer close(r)
		log.Debugln("AutoRelayWithPeerSource called")
		host := hostGetter()
		if host == nil { // context canceled etc.
			return r
		}
		log.Debugln("AutoRelayWithPeerSource called with host")
		log.Debugf("AutoRelayWithPeerSource requested for %d peers\n", numPeers)
		for _, p := range host.Network().Peers() {
			peerProtocols, err := host.Peerstore().GetProtocols(p)
			if err != nil {
				log.Errorln("Error getting peer protocols : ", err)
				continue
			}
			for _, protocol := range peerProtocols {
				if protocol == "/libp2p/circuit/relay/0.2.0/hop" || protocol == "/libp2p/circuit/relay/0.2.0/stop" {
					log.Debugln("AutoRelayWithPeerSource found relay peer")
					select {
					case r <- host.Peerstore().PeerInfo(p):
						log.Debugln("AutoRelayWithPeerSource sent relay peer")
					case <-ctx.Done():
						log.Debugln("AutoRelayWithPeerSource context done")
						return r
					}
				}
			}
		}
		return r
	}
}
