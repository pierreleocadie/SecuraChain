package config

import (
	"fmt"
	"time"
)

const (
	ListeningPortFlag             = 0
	LowWater                      = 160
	HighWater                     = 192
	UserAgent                     = "SecuraChain"
	ProtocolVersion               = "0.0.1"
	RendezvousStringFlag          = "SecuraChainNetwork"
	ClientAnnouncementStringFlag  = "ClientAnnouncement"
	StorageNodeResponseStringFlag = "StorageNodeResponse"
	DHTDiscoveryRefreshInterval   = 10 * time.Second
	FileRights                    = 0700
)

var (
	Ip4tcp         = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", ListeningPortFlag)
	Ip6tcp         = fmt.Sprintf("/ip6/::/tcp/%d", ListeningPortFlag)
	Ip4quic        = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", ListeningPortFlag)
	Ip6quic        = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", ListeningPortFlag)
	BootstrapPeers = []string{
		"/ip4/13.37.148.174/udp/1211/quic-v1/p2p/12D3KooWBm6aEtcGiJNsnsCwaiH4SoqJHZMgvctdQsyAenwyt8Ds",
		"/ip4/154.56.63.167/udp/1211/quic-v1/p2p/12D3KooWA8jcyCRDXhk1H7kgBg1ui1pBi2ezJC4mtmkRuvPFUegc",
	}
)
