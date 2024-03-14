package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
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
	host, err := libp2p.New(
		libp2p.UserAgent("SecuraChain"),
		libp2p.ProtocolVersion("0.0.1"),
		libp2p.ListenAddrStrings(ip4tcp, ip6tcp, ip4quic, ip6quic),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.RandomIdentity,
		libp2p.DefaultSecurity,
		libp2p.DefaultMuxers,
	)
	if err != nil {
		panic(err)
	}
	defer host.Close()
	log.Printf("Our node ID : %s\n", host.ID())

	// Node info
	hostInfo := peer.AddrInfo{
		ID:    host.ID(),
		Addrs: host.Addrs(),
	}

	addrs, err := peer.AddrInfoToP2pAddrs(&hostInfo)
	if err != nil {
		panic(err)
	}

	for _, addr := range addrs {
		log.Println("Node address:", addr)
	}

	for _, addr := range host.Addrs() {
		log.Println("Listening on address:", addr)
	}

	/*
	* NETWORK PEER DISCOVERY WITH DHT
	 */
	// Convert the bootstrap peers from string to multiaddr
	bootstrapPeersStrings := []string{
		"/ip6/2600:3c03::f03c:94ff:fef5:d976/udp/1211/quic-v1/p2p/12D3KooWQp1GawetKx9L2Gq3ZoebZWPeD9AhT6AzxEd3xh36jHbE",
		"/ip4/45.33.86.144/tcp/1211/p2p/12D3KooWQp1GawetKx9L2Gq3ZoebZWPeD9AhT6AzxEd3xh36jHbE",
		"/ip4/45.33.86.144/udp/1211/quic-v1/p2p/12D3KooWQp1GawetKx9L2Gq3ZoebZWPeD9AhT6AzxEd3xh36jHbE",
		"/ip6/2600:3c03::f03c:94ff:fef5:d976/tcp/1211/p2p/12D3KooWQp1GawetKx9L2Gq3ZoebZWPeD9AhT6AzxEd3xh36jHbE",
		"/ip6/2a01:7e02::f03c:94ff:fef5:d991/tcp/1211/p2p/12D3KooWRGhCPo7hRqFZhjMYn4PX6nLXwr3LvuUmKpNwF23QWnDW",
		"/ip6/2a01:7e02::f03c:94ff:fef5:d991/udp/1211/quic-v1/p2p/12D3KooWRGhCPo7hRqFZhjMYn4PX6nLXwr3LvuUmKpNwF23QWnDW",
		"/ip4/172.233.121.214/tcp/1211/p2p/12D3KooWRGhCPo7hRqFZhjMYn4PX6nLXwr3LvuUmKpNwF23QWnDW",
		"/ip4/172.233.121.214/udp/1211/quic-v1/p2p/12D3KooWRGhCPo7hRqFZhjMYn4PX6nLXwr3LvuUmKpNwF23QWnDW",
		"/ip4/172.104.53.15/udp/1211/quic-v1/p2p/12D3KooWCoRYoi19NyY8xoEi4n8bCvVaW8ECUV3Po3EXoHJPiMUo",
		"/ip6/2400:8901::f03c:94ff:fef5:d9e0/tcp/1211/p2p/12D3KooWCoRYoi19NyY8xoEi4n8bCvVaW8ECUV3Po3EXoHJPiMUo",
		"/ip6/2400:8901::f03c:94ff:fef5:d9e0/udp/1211/quic-v1/p2p/12D3KooWCoRYoi19NyY8xoEi4n8bCvVaW8ECUV3Po3EXoHJPiMUo",
		"/ip4/172.104.53.15/tcp/1211/p2p/12D3KooWCoRYoi19NyY8xoEi4n8bCvVaW8ECUV3Po3EXoHJPiMUo",
		"/ip4/172.233.49.69/tcp/1211/p2p/12D3KooWMa9a7r3AXof8TthCk2F5A7hHYUFYLFCeMX8fvnqhMcCy",
		"/ip4/172.233.49.69/udp/1211/quic-v1/p2p/12D3KooWMa9a7r3AXof8TthCk2F5A7hHYUFYLFCeMX8fvnqhMcCy",
		"/ip6/2600:3c0e::f03c:94ff:fef5:d955/tcp/1211/p2p/12D3KooWMa9a7r3AXof8TthCk2F5A7hHYUFYLFCeMX8fvnqhMcCy",
		"/ip6/2600:3c0e::f03c:94ff:fef5:d955/udp/1211/quic-v1/p2p/12D3KooWMa9a7r3AXof8TthCk2F5A7hHYUFYLFCeMX8fvnqhMcCy",
		"/ip6/2a01:7e00::f03c:94ff:fef5:d95d/udp/1211/quic-v1/p2p/12D3KooWGPQzyfJgUqeF7rL5v1hFVC4zVR7rCX7Yn9HmE8AL2ebM",
		"/ip4/212.111.43.113/tcp/1211/p2p/12D3KooWGPQzyfJgUqeF7rL5v1hFVC4zVR7rCX7Yn9HmE8AL2ebM",
		"/ip4/212.111.43.113/udp/1211/quic-v1/p2p/12D3KooWGPQzyfJgUqeF7rL5v1hFVC4zVR7rCX7Yn9HmE8AL2ebM",
		"/ip6/2a01:7e00::f03c:94ff:fef5:d95d/tcp/1211/p2p/12D3KooWGPQzyfJgUqeF7rL5v1hFVC4zVR7rCX7Yn9HmE8AL2ebM",
		"/ip4/172.235.26.215/tcp/1211/p2p/12D3KooWLoZqrQ8rcmCkY1ngcaAxpbo4DX2vm8WdkHDcxbAuv1VZ",
		"/ip4/172.235.26.215/udp/1211/quic-v1/p2p/12D3KooWLoZqrQ8rcmCkY1ngcaAxpbo4DX2vm8WdkHDcxbAuv1VZ",
		"/ip6/2600:3c08::f03c:94ff:fef5:d995/tcp/1211/p2p/12D3KooWLoZqrQ8rcmCkY1ngcaAxpbo4DX2vm8WdkHDcxbAuv1VZ",
		"/ip6/2600:3c08::f03c:94ff:fef5:d995/udp/1211/quic-v1/p2p/12D3KooWLoZqrQ8rcmCkY1ngcaAxpbo4DX2vm8WdkHDcxbAuv1VZ",
		"/ip4/172.105.104.135/tcp/1211/p2p/12D3KooWEkrT28UjpX8ywnYZ6MSaKmWFYo8zAjxBAB4XVkGiSo9x",
		"/ip4/172.105.104.135/udp/1211/quic-v1/p2p/12D3KooWEkrT28UjpX8ywnYZ6MSaKmWFYo8zAjxBAB4XVkGiSo9x",
		"/ip6/2600:3c04::f03c:94ff:fef5:d900/tcp/1211/p2p/12D3KooWEkrT28UjpX8ywnYZ6MSaKmWFYo8zAjxBAB4XVkGiSo9x",
		"/ip6/2600:3c04::f03c:94ff:fef5:d900/udp/1211/quic-v1/p2p/12D3KooWEkrT28UjpX8ywnYZ6MSaKmWFYo8zAjxBAB4XVkGiSo9x",
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
	if err := dhtDiscovery.Run(ctx, host); err != nil {
		log.Fatalf("Failed to run DHT: %s", err)
	}

	/*
	* NETWORK PEER DISCOVERY WITH mDNS
	 */
	// Initialize mDNS
	mdnsConfig := netwrk.NewMDNSDiscovery(*rendezvousStringFlag)

	// Run MDNS
	mdnsConfig.Run(host)

	/*
	* PUBLISH AND SUBSCRIBE TO TOPICS
	 */
	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, host)
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
			if msg.GetFrom().String() == host.ID().String() {
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
			err := keepRelayConnectionAliveTopic.Publish(ctx, netwrk.GeneratePacket(host.ID()))
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

		time.Sleep(2 * time.Minute)

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
				if msg.ReceivedFrom == host.ID() {
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
	subNet, err := host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
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
