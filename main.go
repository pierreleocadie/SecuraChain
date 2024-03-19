package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	ipfsLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	netwrk "github.com/pierreleocadie/SecuraChain/internal/network"
	"github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
	poccontext "github.com/pierreleocadie/SecuraChain/poc-context"
)

const RefreshInterval = 10 * time.Second

var (
	rendezvousStringFlag *string = flag.String("rendezvousString", "SecuraChain", "Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	networkRoleFlag      *string = flag.String("networkRole", os.Args[1], "network role of the node")
	ip4tcp               string  = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 0)
	ip6tcp               string  = fmt.Sprintf("/ip6/::/tcp/%d", 0)
	ip4quic              string  = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", 0)
	ip6quic              string  = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", 0)
)

func main() {
	logg := ipfsLog.Logger("test-node")
	err := ipfsLog.SetLogLevel("test-node", "DEBUG")
	if err != nil {
		log.Println("Error setting log level : ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	minerKeyPair, err := ecdsa.NewECDSAKeyPair()
	if err != nil {
		log.Fatalln("Failed to create ECDSA key pair:", err)
		return
	}

	/*
	* ROLE VALIDATION
	 */
	if *networkRoleFlag != "storer" && *networkRoleFlag != "miner" && *networkRoleFlag != "chainer" {
		log.Fatalln("Invalid network role")
		return
	}

	/*
	* NODE INITIALIZATION
	 */
	var h host.Host
	hostReady := make(chan struct{})
	hostGetter := func() host.Host {
		<-hostReady // closed when we finish setting up the host
		return h
	}

	h, err = libp2p.New(
		libp2p.UserAgent("SecuraChain"),
		libp2p.ProtocolVersion("0.0.1"),
		libp2p.AddrsFactory(netwrk.FilterOutPrivateAddrs), // Comment this line to build bootstrap node
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.ListenAddrStrings(ip4tcp, ip6tcp, ip4quic, ip6quic),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.RandomIdentity,
		libp2p.DefaultSecurity,
		libp2p.DefaultMuxers,
		libp2p.DefaultEnableRelay,
		libp2p.EnableRelayService(),
		libp2p.EnableAutoRelayWithPeerSource(
			netwrk.NewPeerSource(logg, hostGetter),
			autorelay.WithBackoff(10*time.Second),
			autorelay.WithMinInterval(10*time.Second),
			autorelay.WithNumRelays(1),
			autorelay.WithMinCandidates(1),
		),
	)
	if err != nil {
		panic(err)
	}
	defer h.Close()
	log.Printf("Our node ID : %s\n", h.ID())

	close(hostReady)

	// Node info
	hostInfo := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}

	addrs, err := peer.AddrInfoToP2pAddrs(&hostInfo)
	if err != nil {
		panic(err)
	}

	for _, addr := range addrs {
		log.Println("Node address:", addr)
	}

	for _, addr := range h.Addrs() {
		log.Println("Listening on address:", addr)
	}

	/*
	* RELAY SERVICE
	 */
	// Check if the node is behind NAT
	behindNAT := netwrk.NATDiscovery(logg)

	// If the node is behind NAT, search for a node that supports relay
	// TODO: Optimize this code
	if !behindNAT {
		log.Println("Node is not behind NAT")
		// Start the relay service
		_, err = relay.New(h,
			relay.WithResources(relay.Resources{
				Limit: &relay.RelayLimit{
					// Duration is the time limit before resetting a relayed connection; defaults to 2min.
					Duration: 1 * time.Hour,
					// Data is the limit of data relayed (on each direction) before resetting the connection.
					// Defaults to 128KB
					// Data: 1 << 30, // 1 GB
					Data: 1 << 30,
				},
				// Default values
				// ReservationTTL is the duration of a new (or refreshed reservation).
				// Defaults to 1hr.
				ReservationTTL: 1 * time.Hour,
				// MaxReservations is the maximum number of active relay slots; defaults to 128.
				MaxReservations: 128,
				// MaxCircuits is the maximum number of open relay connections for each peer; defaults to 16.
				MaxCircuits: 16,
				// BufferSize is the size of the relayed connection buffers; defaults to 2048.
				BufferSize: 2048,

				// MaxReservationsPerPeer is the maximum number of reservations originating from the same
				// peer; default is 4.
				MaxReservationsPerPeer: 4,
				// MaxReservationsPerIP is the maximum number of reservations originating from the same
				// IP address; default is 8.
				MaxReservationsPerIP: 8,
				// MaxReservationsPerASN is the maximum number of reservations origination from the same
				// ASN; default is 32
				MaxReservationsPerASN: 32,
			}),
		)
		if err != nil {
			log.Println("Error instantiating relay service : ", err)
		}
		log.Println("Relay service started")
	} else {
		log.Println("Node is behind NAT")
	}

	/*
	* NETWORK PEER DISCOVERY WITH DHT
	 */
	// Convert the bootstrap peers from string to multiaddr
	bootstrapPeersStrings := []string{
		"/ip4/172.233.121.117/tcp/1211/p2p/12D3KooWCHGr2xHYFBcEPKbdp1NFHciYSXc7BiRLHfXKWsedaGo8",
		"/ip4/172.233.121.117/udp/1211/quic-v1/p2p/12D3KooWCHGr2xHYFBcEPKbdp1NFHciYSXc7BiRLHfXKWsedaGo8",
		"/ip6/2a01:7e02::f03c:94ff:fe40:65d/tcp/1211/p2p/12D3KooWCHGr2xHYFBcEPKbdp1NFHciYSXc7BiRLHfXKWsedaGo8",
		"/ip6/2a01:7e02::f03c:94ff:fe40:65d/udp/1211/quic-v1/p2p/12D3KooWCHGr2xHYFBcEPKbdp1NFHciYSXc7BiRLHfXKWsedaGo8",
		"/ip4/97.107.135.203/tcp/1211/p2p/12D3KooWQM38Zi3dhcMXsnDz5G3hD5C7a72tWUHcNHZ3xNNxZdYD",
		"/ip4/97.107.135.203/udp/1211/quic-v1/p2p/12D3KooWQM38Zi3dhcMXsnDz5G3hD5C7a72tWUHcNHZ3xNNxZdYD",
		"/ip6/2600:3c03::f03c:94ff:fe40:69f/tcp/1211/p2p/12D3KooWQM38Zi3dhcMXsnDz5G3hD5C7a72tWUHcNHZ3xNNxZdYD",
		"/ip6/2600:3c03::f03c:94ff:fe40:69f/udp/1211/quic-v1/p2p/12D3KooWQM38Zi3dhcMXsnDz5G3hD5C7a72tWUHcNHZ3xNNxZdYD",
		"/ip4/172.105.105.151/tcp/1211/p2p/12D3KooWJV4ixBevkwSypVtiCLvMaS981hqNqh82MoMxFVdNsTSe",
		"/ip4/172.105.105.151/udp/1211/quic-v1/p2p/12D3KooWJV4ixBevkwSypVtiCLvMaS981hqNqh82MoMxFVdNsTSe",
		"/ip6/2600:3c04::f03c:94ff:fe40:67e/tcp/1211/p2p/12D3KooWJV4ixBevkwSypVtiCLvMaS981hqNqh82MoMxFVdNsTSe",
		"/ip6/2600:3c04::f03c:94ff:fe40:67e/udp/1211/quic-v1/p2p/12D3KooWJV4ixBevkwSypVtiCLvMaS981hqNqh82MoMxFVdNsTSe",
		"/ip4/172.105.177.169/udp/1211/quic-v1/p2p/12D3KooWGAF6ujQDTsXV51j53c7PWB7JG6pJW5KY9TsqF5iwnX8d",
		"/ip6/2400:8907::f03c:94ff:fe40:6c1/tcp/1211/p2p/12D3KooWGAF6ujQDTsXV51j53c7PWB7JG6pJW5KY9TsqF5iwnX8d",
		"/ip6/2400:8907::f03c:94ff:fe40:6c1/udp/1211/quic-v1/p2p/12D3KooWGAF6ujQDTsXV51j53c7PWB7JG6pJW5KY9TsqF5iwnX8d",
		"/ip4/172.105.177.169/tcp/1211/p2p/12D3KooWGAF6ujQDTsXV51j53c7PWB7JG6pJW5KY9TsqF5iwnX8d",
		"/ip4/172.232.115.247/tcp/1211/p2p/12D3KooWGmcjHGjFvQGKEoSqfx1xgqapQ85oNwgCjjrSDM3aZfCh",
		"/ip4/172.232.115.247/udp/1211/quic-v1/p2p/12D3KooWGmcjHGjFvQGKEoSqfx1xgqapQ85oNwgCjjrSDM3aZfCh",
		"/ip6/2600:3c08::f03c:94ff:fe40:65e/tcp/1211/p2p/12D3KooWGmcjHGjFvQGKEoSqfx1xgqapQ85oNwgCjjrSDM3aZfCh",
		"/ip6/2600:3c08::f03c:94ff:fe40:65e/udp/1211/quic-v1/p2p/12D3KooWGmcjHGjFvQGKEoSqfx1xgqapQ85oNwgCjjrSDM3aZfCh",
		"/ip4/139.162.192.195/tcp/1211/p2p/12D3KooWSYwnofKMaCoFJW268LrRdqns89PMS3fESY2Un8rh8RPp",
		"/ip4/139.162.192.195/udp/1211/quic-v1/p2p/12D3KooWSYwnofKMaCoFJW268LrRdqns89PMS3fESY2Un8rh8RPp",
		"/ip6/2a01:7e00::f03c:94ff:fe40:610/tcp/1211/p2p/12D3KooWSYwnofKMaCoFJW268LrRdqns89PMS3fESY2Un8rh8RPp",
		"/ip6/2a01:7e00::f03c:94ff:fe40:610/udp/1211/quic-v1/p2p/12D3KooWSYwnofKMaCoFJW268LrRdqns89PMS3fESY2Un8rh8RPp",
		"/ip4/172.233.27.107/tcp/1211/p2p/12D3KooWKkzHApqQz1mC6Cy61fuUFi3L6wp5UVeVM6CrvR5ePJ1j",
		"/ip4/172.233.27.107/udp/1211/quic-v1/p2p/12D3KooWKkzHApqQz1mC6Cy61fuUFi3L6wp5UVeVM6CrvR5ePJ1j",
		"/ip6/2600:3c0d::f03c:94ff:fe40:62d/tcp/1211/p2p/12D3KooWKkzHApqQz1mC6Cy61fuUFi3L6wp5UVeVM6CrvR5ePJ1j",
		"/ip6/2600:3c0d::f03c:94ff:fe40:62d/udp/1211/quic-v1/p2p/12D3KooWKkzHApqQz1mC6Cy61fuUFi3L6wp5UVeVM6CrvR5ePJ1j",
		"/ip4/172.233.61.106/tcp/1211/p2p/12D3KooWJH1Db3B7meWR1UrLJByfKqk4TWurKXWprREpKyfo5doy",
		"/ip4/172.233.61.106/udp/1211/quic-v1/p2p/12D3KooWJH1Db3B7meWR1UrLJByfKqk4TWurKXWprREpKyfo5doy",
		"/ip6/2600:3c0e::f03c:94ff:fe40:6cb/tcp/1211/p2p/12D3KooWJH1Db3B7meWR1UrLJByfKqk4TWurKXWprREpKyfo5doy",
		"/ip6/2600:3c0e::f03c:94ff:fe40:6cb/udp/1211/quic-v1/p2p/12D3KooWJH1Db3B7meWR1UrLJByfKqk4TWurKXWprREpKyfo5doy",
	}
	var bootstrapPeersMultiaddr []multiaddr.Multiaddr
	for _, peer := range bootstrapPeersStrings {
		peerMultiaddr, err := multiaddr.NewMultiaddr(peer)
		if err != nil {
			log.Println("Error converting bootstrap peer to multiaddr : ", err)
		}
		bootstrapPeersMultiaddr = append(bootstrapPeersMultiaddr, peerMultiaddr)
	}

	// Initialize DHT in server mode
	dhtDiscovery := netwrk.NewDHTDiscovery(
		false,
		"SecuraChainNetwork",
		bootstrapPeersMultiaddr,
		10*time.Second,
	)

	// Run DHT
	if err := dhtDiscovery.Run(ctx, h); err != nil {
		log.Fatalf("Failed to run DHT: %s", err)
	}

	/*
	* NETWORK PEER DISCOVERY WITH mDNS
	 */
	// Initialize mDNS
	mdnsConfig := netwrk.NewMDNSDiscovery(*rendezvousStringFlag)

	// Run MDNS
	mdnsConfig.Run(h)

	/*
	* PUBLISH AND SUBSCRIBE TO TOPICS
	 */
	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Println("Failed to create new PubSub service:", err)
	}

	// KeepRelayConnectionAlive
	keepRelayConnectionAliveTopic, err := ps.Join("KeepRelayConnectionAlive")
	if err != nil {
		log.Printf("Failed to join KeepRelayConnectionAlive topic: %s", err)
	}

	// Subscribe to KeepRelayConnectionAlive topic
	subKeepRelayConnectionAlive, err := keepRelayConnectionAliveTopic.Subscribe()
	if err != nil {
		log.Printf("Failed to subscribe to KeepRelayConnectionAlive topic: %s", err)
	}

	// Handle incoming KeepRelayConnectionAlive messages
	go func() {
		for {
			msg, err := subKeepRelayConnectionAlive.Next(ctx)
			if err != nil {
				log.Printf("Failed to get next message from KeepRelayConnectionAlive topic: %s", err)
				continue
			}
			if msg.GetFrom().String() == h.ID().String() {
				continue
			}
			log.Printf("Received KeepRelayConnectionAlive message from %s", msg.GetFrom().String())
			log.Printf("KeepRelayConnectionAlive: %s", string(msg.Data))
		}
	}()

	// Handle outgoing KeepRelayConnectionAlive messages
	go func() {
		for {
			time.Sleep(15 * time.Second)
			err := keepRelayConnectionAliveTopic.Publish(ctx, netwrk.GeneratePacket(h.ID()))
			if err != nil {
				log.Printf("Failed to publish KeepRelayConnectionAlive message: %s", err)
				continue
			}
			log.Println("KeepRelayConnectionAlive message sent successfully")
		}
	}()

	if *networkRoleFlag == "storer" {
		// Join the NewTransaction topic and generate fake transaction every random time between 5 and 10s
		newTransactionTopic, err := ps.Join("NewTransaction")
		if err != nil {
			log.Panicf("Failed to join NewTransaction topic: %s", err)
		}

		go func() {
			for {
				time.Sleep(5 * time.Second)
				fakeTrx, err := poccontext.GenFakeTransaction()
				if err != nil {
					log.Println("Failed to generate fake transaction:", err)
					continue
				}
				serializedTrx, err := fakeTrx.Serialize()
				if err != nil {
					log.Println("Failed to serialize fake transaction:", err)
					continue
				}
				if err = newTransactionTopic.Publish(ctx, serializedTrx); err != nil {
					log.Println("Failed to publish fake transaction:", err)
				}
			}
		}()
	} else if *networkRoleFlag == "miner" {
		// Join the NewBlock topic
		newBlockTopic, err := ps.Join("BlockAnnouncement")
		if err != nil {
			log.Panicf("Failed to join NewBlock topic: %s", err)
		}

		subNewBlock, err := newBlockTopic.Subscribe()
		if err != nil {
			log.Panicf("Failed to subscribe to NewBlock topic: %s", err)
		}

		// Subscribe to NewTransaction topic
		newTransactionTopic, err := ps.Join("NewTransaction")
		if err != nil {
			log.Panicf("Failed to subscribe to NewTransaction topic: %s", err)
		}

		subNewTransaction, err := newTransactionTopic.Subscribe()
		if err != nil {
			log.Panicf("Failed to subscribe to NewTransaction topic: %s", err)
		}

		time.Sleep(1 * time.Minute)
		time.Sleep(30 * time.Second)

		trxPool := []transaction.Transaction{}

		// Handle NewTransaction events in a separate goroutine
		// Add new transactions to the transaction pool
		go func() {
			for {
				msg, err := subNewTransaction.Next(ctx)
				if err != nil {
					log.Println("Failed to get next message from NewTransaction topic:", err)
					continue
				}
				transactionFactory := transaction.AddFileTransactionFactory{}
				trx, err := transaction.DeserializeTransaction(msg.Data, transactionFactory)
				if err != nil {
					log.Println("Failed to deserialize transaction:", err)
					continue
				}
				if !consensus.ValidateTransaction(trx) {
					log.Println("Invalid transaction")
					continue
				}
				trxPool = append(trxPool, trx)
				log.Println("Transaction added to pool")
				log.Println("Transaction pool size:", len(trxPool))
			}
		}()

		blockPool := []*block.Block{}
		var previousBlock *block.Block = nil
		currentBlock := block.NewBlock(trxPool, nil, 1, minerKeyPair)
		go func() {
			for {
				log.Println("MINING A NEW BLOCK")
				consensus.MineBlock(currentBlock)
				err = currentBlock.SignBlock(minerKeyPair)
				if err != nil {
					log.Println("Failed to sign block:", err)
					continue
				}
				if !consensus.ValidateBlock(currentBlock, previousBlock) {
					log.Println("Invalid block")
					continue
				}
				blockPool = append(blockPool, currentBlock)
				log.Println("Block added to pool")
				log.Println("Block pool size:", len(blockPool))
				previousBlock = currentBlock
				serializedBlock, err := currentBlock.Serialize()
				if err != nil {
					log.Println("Failed to serialize block:", err)
					continue
				}
				if err = newBlockTopic.Publish(ctx, serializedBlock); err != nil {
					log.Println("Failed to publish block:", err)
				}
				// Add the block to a file and append each new block to the file
				// testFile, err := os.OpenFile("./temp/testFile.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				// if err != nil {
				// 	log.Println("Failed to open file:", err)
				// 	continue
				// }
				// serializedBlock, err := currentBlock.Serialize()
				// if err != nil {
				// 	log.Println("Failed to serialize block:", err)
				// 	continue
				// }
				// _, err = testFile.Write(serializedBlock)
				// if err != nil {
				// 	log.Println("Failed to write block to file:", err)
				// 	continue
				// }
				// testFile.Close()
				// log.Println("Block written to file")
				currentBlock = block.NewBlock(trxPool, block.ComputeHash(previousBlock), previousBlock.Header.Height+1, minerKeyPair)
				trxPool = []transaction.Transaction{}
			}
		}()

		// Handle NewBlock events in a separate goroutine
		go func() {
			for {
				msg, err := subNewBlock.Next(ctx)
				if err != nil {
					log.Println("Failed to get next message from NewBlock topic:", err)
					continue
				}
				if msg.ReceivedFrom == h.ID() {
					continue
				}
				newBlock, err := block.DeserializeBlock(msg.Data)
				if err != nil {
					log.Println("Failed to deserialize block:", err)
					continue
				}
				if consensus.ValidateBlock(newBlock, previousBlock) {
					blockPool = append(blockPool, newBlock)
					log.Println("Block added to pool")
					log.Println("Block pool size:", len(blockPool))
					previousBlock = newBlock
				}
			}
		}()
	}

	/*
	* DISPLAY PEER CONNECTEDNESS CHANGES
	 */
	// Subscribe to EvtPeerConnectednessChanged events
	subNet, err := h.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Println("Failed to subscribe to EvtPeerConnectednessChanged:", err)
	}
	defer subNet.Close()

	// Handle connection events in a separate goroutine
	go func() {
		for e := range subNet.Out() {
			evt := e.(event.EvtPeerConnectednessChanged)
			if evt.Connectedness == network.Connected {
				log.Println("Peer connected:", evt.Peer)
			} else if evt.Connectedness == network.NotConnected {
				log.Println("Peer disconnected:", evt.Peer)
			}
		}
	}()

	select {}
}
