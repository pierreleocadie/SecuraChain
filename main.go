package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
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
	ecdsaSC "github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

const RefreshInterval = 10 * time.Second

var (
	ignoredPeers             map[peer.ID]bool = make(map[peer.ID]bool)
	rendezvousStringFlag     *string          = flag.String("rendezvousString", "SecuraChain", "Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	networkRoleFlag          *string          = flag.String("networkRole", os.Args[1], "network role of the node")
	transactionTopicNameFlag *string          = flag.String("transactionTopicName", "NewTransaction", "name of topic to join")
	blockTopicNameFlag       *string          = flag.String("blockTopicName", "NewBlock", "name of topic to join")
	chainTopicNameFlag       *string          = flag.String("chainTopicName", "NewChainVersion", "name of topic to join")
	ip4tcp                   string           = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 0)
	ip6tcp                   string           = fmt.Sprintf("/ip6/::/tcp/%d", 0)
	ip4quic                  string           = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", 0)
	ip6quic                  string           = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", 0)
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	/*
	* ROLE VALIDATION
	 */
	if *networkRoleFlag != "storer" && *networkRoleFlag != "miner" && *networkRoleFlag != "chainer" {
		log.Fatalln("Invalid network role")
		return
	}

	/*
	* GENERATE ECDSA KEY PAIR FOR NODE IDENTITY
	 */
	// Generate a pair of ecdsa keys
	keyPair, err := ecdsaSC.NewECDSAKeyPair()
	if err != nil {
		log.Println("Failed to generate ecdsa key pair:", err)
	}

	/*
	* NODE INITIALIZATION
	 */
	host, err := libp2p.New(
		libp2p.UserAgent("SecuraChain"),
		libp2p.ProtocolVersion("0.0.1"),
		libp2p.EnableHolePunching(),
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
	// Initialize DHT
	dhtConfig := &DHT{
		RendezvousString:          rendezvousStringFlag,
		DiscorveryRefreshInterval: RefreshInterval,
		Bootstrap:                 true,
	}
	if len(os.Args) > 2 {
		bootstrapPeer, _ := multiaddr.NewMultiaddr(os.Args[2])
		dhtConfig = &DHT{
			RendezvousString:          rendezvousStringFlag,
			BootstrapPeers:            []multiaddr.Multiaddr{bootstrapPeer},
			DiscorveryRefreshInterval: RefreshInterval,
			Bootstrap:                 false,
		}
	}

	// Run DHT
	if err := dhtConfig.Run(ctx, host); err != nil {
		log.Fatal("Failed to run DHT:", err)
	}

	/*
	* PUBLISH AND SUBSCRIBE TO TOPICS
	 */
	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Println("Failed to create new PubSub service:", err)
	}

	if *networkRoleFlag == "storer" {
		// Join the topic
		topicTrx, err := ps.Join(*transactionTopicNameFlag)
		if err != nil {
			log.Println("Failed to join topic:", err)
		}

		subTrx, err := topicTrx.Subscribe()
		if err != nil {
			log.Println("Failed to subscribe to topic:", err)
		}

		// Every X seconds, publish a new transaction - random interval between 1 and 10 seconds
		go func() {
			for {
				time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
				// Publish the transaction
				if err := topicTrx.Publish(ctx, generateTransaction(host.ID())); err != nil {
					log.Println("Failed to publish transaction:", err)
				}
			}
		}()

		// Handle incoming transactions in a separate goroutine
		go func() {
			for {
				msg, err := subTrx.Next(ctx)
				if err != nil {
					log.Println("Failed to get next transaction:", err)
				}
				log.Println(string(msg.Data))
			}
		}()
	} else if *networkRoleFlag == "miner" {
		// Join the topic for transactions
		topicTrx, err := ps.Join(*transactionTopicNameFlag)
		if err != nil {
			log.Println("Failed to join topic:", err)
		}

		// Join the topic for blocks
		topicBlock, err := ps.Join(*blockTopicNameFlag)
		if err != nil {
			log.Println("Failed to join topic:", err)
		}

		// Subscribe to the topic for transactions
		subTrx, err := topicTrx.Subscribe()
		if err != nil {
			log.Println("Failed to subscribe to topic:", err)
		}

		// Subscribe to the topic for blocks
		subBlock, err := topicBlock.Subscribe()
		if err != nil {
			log.Println("Failed to subscribe to topic:", err)
		}

		// Define a transaction pool
		var trxPool []Transaction

		// Every X seconds, generate a new block - random interval between 1 and 10 seconds
		go func() {
			for {
				time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
				// Generate the block
				block := generateBlock(&trxPool, host.ID(), keyPair)
				if block != nil {
					// Publish the block
					if err := topicBlock.Publish(ctx, block); err != nil {
						log.Println("Failed to publish block:", err)
					}
				}
			}
		}()

		// Handle incoming transactions in a separate goroutine
		go func() {
			for {
				msg, err := subTrx.Next(ctx)
				if err != nil {
					log.Println("Failed to get next transaction:", err)
				}
				// Add the transaction to the pool
				addTrxToPool(msg.Data, &trxPool)
			}
		}()

		// Handle incoming blocks in a separate goroutine
		go func() {
			for {
				msg, err := subBlock.Next(ctx)
				if err != nil {
					log.Println("Failed to get next block:", err)
				}
				receivedBlock := Block{}
				e := json.Unmarshal(msg.Data, &receivedBlock)
				if e != nil {
					log.Printf("Error unmarshalling block: %v", err)
					return
				}
				if ValidateBlock(msg.Data) {
					if receivedBlock.Header.MinedBy == host.ID() {
						log.Println("Valid block emitted")
					} else {
						log.Println("Valid block received")
					}
					resetPool(&trxPool)
				} else {
					if receivedBlock.Header.MinedBy == host.ID() {
						log.Println("Invalid block emitted")
					} else {
						log.Println("Invalid block received")
					}
				}
			}
		}()
	} else if *networkRoleFlag == "chainer" {
		// Join the topic for blocks
		topicBlock, err := ps.Join(*blockTopicNameFlag)
		if err != nil {
			log.Println("Failed to join topic:", err)
		}

		// Join the topic for chain versions
		topicChain, err := ps.Join(*chainTopicNameFlag)
		if err != nil {
			log.Println("Failed to join topic:", err)
		}

		// Subscribe to the topic for blocks
		subBlock, err := topicBlock.Subscribe()
		if err != nil {
			log.Println("Failed to subscribe to topic:", err)
		}

		// Subscribe to the topic for chain versions
		subChain, err := topicChain.Subscribe()
		if err != nil {
			log.Println("Failed to subscribe to topic:", err)
		}

		// Define a chain
		var chain *Chain = &Chain{
			ChainVersion: 1,
			Chain:        []Block{},
		}

		// Handle incoming blocks in a separate goroutine
		go func() {
			for {
				msg, err := subBlock.Next(ctx)
				if err != nil {
					log.Println("Failed to get next block:", err)
				}
				if ValidateBlock(msg.Data) {
					// Add the block to the chain
					addBlockToChain(msg.Data, chain)
					b, _ := json.Marshal(chain)
					fmt.Println(string(b))
				} else {
					log.Println("Invalid block received")
				}
			}
		}()

		// Handle incoming chain versions in a separate goroutine
		go func() {
			for {
				msg, err := subChain.Next(ctx)
				if err != nil {
					log.Println("Failed to get next chain version:", err)
				}
				log.Println(string(msg.Data))
			}
		}()

		// Every X seconds, publish a new chain version - random interval between 1 and 10 seconds - To sync the chain all over the network
		/* go func() {
			for {
				time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
				// Publish the chain version
				if err := topicChain.Publish(ctx, []byte(chain)); err != nil {
					log.Println("Failed to publish chain version:", err)
				}
			}
		}() */
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
