package config

import (
	"fmt"
	"time"
)

const (
	ListeningPortFlag             = 0 // 0 means random port, 1211 is the port for the bootstrap node, 1212 is the port for bootstrap node test
	LowWater                      = 160
	HighWater                     = 192
	UserAgent                     = "SecuraChain"
	ProtocolVersion               = "0.0.1"
	RendezvousStringFlag          = "SecuraChainNetwork"
	ClientAnnouncementStringFlag  = "ClientAnnouncement"
	StorageNodeResponseStringFlag = "StorageNodeResponse"
	DHTDiscoveryRefreshInterval   = 10 * time.Second
	FileRights                    = 0700
	FileMetadataRegistryJSON      = "fileMetadataRegistry.json"
	PeerstoreCleanupInterval      = 1 * time.Minute
)

var (
	IP4tcp         = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", ListeningPortFlag)
	IP6tcp         = fmt.Sprintf("/ip6/::/tcp/%d", ListeningPortFlag)
	IP4quic        = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", ListeningPortFlag)
	IP6quic        = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", ListeningPortFlag)
	BootstrapPeers = []string{
		"/ip4/13.37.148.174/tcp/1212/p2p/12D3KooWSwK3gCNp3pc7MQYsF4giuk78wPEGwg5zSGQRXHgrgkan",         // bootstrap node test
		"/ip4/13.37.148.174/udp/1212/quic-v1/p2p/12D3KooWSwK3gCNp3pc7MQYsF4giuk78wPEGwg5zSGQRXHgrgkan", // bootstrap node test
		// "/ip4/13.37.148.174/tcp/1211/p2p/12D3KooWBm6aEtcGiJNsnsCwaiH4SoqJHZMgvctdQsyAenwyt8Ds",
		// "/ip4/13.37.148.174/udp/1211/quic-v1/p2p/12D3KooWBm6aEtcGiJNsnsCwaiH4SoqJHZMgvctdQsyAenwyt8Ds",
		// "/ip4/154.56.63.167/tcp/1211/p2p/12D3KooWA8jcyCRDXhk1H7kgBg1ui1pBi2ezJC4mtmkRuvPFUegc",
		// "/ip4/154.56.63.167/udp/1211/quic-v1/p2p/12D3KooWA8jcyCRDXhk1H7kgBg1ui1pBi2ezJC4mtmkRuvPFUegc",
	}
)
