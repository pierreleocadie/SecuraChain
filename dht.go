// Major parts of this code is taken from : https://github.com/mudler/edgevpn/blob/master/pkg/discovery/dht.go
package main

import (
	"context"
	"log"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
)

type DHT struct {
	RendezvousString          *string
	BootstrapPeers            []multiaddr.Multiaddr
	DiscorveryRefreshInterval time.Duration
	Bootstrap                 bool
	*dht.IpfsDHT              // Embed the IpfsDHT -> DHT will implement all the methods of IpfsDHT, its like inheritance
}

func (d *DHT) startDHT(ctx context.Context, host host.Host) (*dht.IpfsDHT, error) {
	if d.IpfsDHT == nil {
		if !d.Bootstrap {
			kademliaDHT, err := dht.New(ctx, host, dht.MaxRecordAge(10*time.Second))
			if err != nil {
				return d.IpfsDHT, err
			}
			d.IpfsDHT = kademliaDHT
			return d.IpfsDHT, nil
		} else {
			kademliaDHT, err := dht.New(ctx, host, dht.Mode(dht.ModeServer))
			if err != nil {
				return d.IpfsDHT, err
			}
			d.IpfsDHT = kademliaDHT
			return d.IpfsDHT, nil
		}
	}
	return d.IpfsDHT, nil
}

func (d *DHT) bootstrapPeers(ctx context.Context, host host.Host) {
	var wg sync.WaitGroup
	for _, peerAddr := range d.BootstrapPeers {
		peerInfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if host.Network().Connectedness(peerInfo.ID) != network.Connected {
				if err := host.Connect(ctx, *peerInfo); err != nil {
					log.Println("Connection failed")
					// log.Println("Connection failed:", err)
				} else {
					log.Println("[bootstrapPeers] Connection success:", peerInfo.ID)
				}
			}
		}()
	}
	wg.Wait()
}

func (d *DHT) AnnounceAndConnect(ctx context.Context, host host.Host, routingDiscovery *drouting.RoutingDiscovery) {
	_, err := routingDiscovery.Advertise(ctx, *d.RendezvousString)
	if err != nil {
		log.Println("[AnnounceAndConnect] Error announcing self : ", err)
	} else {
		log.Println("[AnnounceAndConnect] Successfully announced!")
	}

	log.Println("[AnnounceAndConnect] Searching for other peers...")

	peersChan, err := routingDiscovery.FindPeers(ctx, *d.RendezvousString)
	if err != nil {
		log.Println("[AnnounceAndConnect] Error finding peers : ", err)
	}

	peersCount := 0
	for p := range peersChan {
		// Don't dial ourselves or peers without address
		if p.ID == host.ID() || len(p.Addrs) == 0 {
			continue
		}

		// Ignore peers that are in the ignored list
		if _, ok := ignoredPeers[p.ID]; ok {
			continue
		}

		if host.Network().Connectedness(p.ID) != network.Connected {
			peersCount++

			log.Println("[announceAndConnect] Found peer:", p)

			if err := host.Connect(ctx, p); err != nil {
				log.Println("[AnnounceAndConnect] Connection failed")
				log.Println("[AnnounceAndConnect] Connection failed : ", err)
				ignoredPeers[p.ID] = true
			} else {
				log.Println("[AnnounceAndConnect] Connection success : ", p)
			}
		}
	}
	log.Println("[announceAndConnect] Number of peers found : ", peersCount)
}

func (d *DHT) Run(ctx context.Context, host host.Host) error {
	kademliaDHT, err := d.startDHT(ctx, host)
	if err != nil {
		log.Fatal(err)
		return err
	}

	if d.Bootstrap {
		log.Print("[Run] BOOTSTRAP NODE - DHT IN SERVER MODE")
	} else {
		log.Println("[Run] Bootstrapping DHT")
	}

	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return err
	}

	if !d.Bootstrap {
		d.bootstrapPeers(ctx, host)

		log.Println("[Run] Announcing ourselves...")
		log.Println("[Run] Rendezvous point:", *d.RendezvousString)
		routingDiscovery := drouting.NewRoutingDiscovery(d.IpfsDHT)

		connect := func() {
			d.AnnounceAndConnect(ctx, host, routingDiscovery)
		}

		go func() {
			connect()
			t := time.NewTicker(d.DiscorveryRefreshInterval)
			defer t.Stop()
			for {
				select {
				case <-t.C:
					connect()
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	return nil
}
