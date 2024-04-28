package ipfs

import (
	"context"
	"sync"

	ipfsLog "github.com/ipfs/go-log/v2"
	icore "github.com/ipfs/kubo/core/coreiface"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// This function allows a node to connect to other peers given in an IPFS network
func ConnectToPeers(log *ipfsLog.ZapEventLogger, ctx context.Context, ipfs icore.CoreAPI, peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peer.AddrInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peer.AddrInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}
	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peer.AddrInfo) {
			defer wg.Done()
			err := ipfs.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				log.Warnln("failed to connect to %s: %s", peerInfo.ID, err)
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}
