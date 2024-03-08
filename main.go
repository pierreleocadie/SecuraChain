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
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
	"github.com/pierreleocadie/SecuraChain/internal/core/consensus"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	mdns "github.com/pierreleocadie/SecuraChain/internal/network"
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
	* NETWORK PEER DISCOVERY WITH mDNS
	 */
	// Initialize mDNS
	mdnsConfig := mdns.NewMDNSDiscovery(*rendezvousStringFlag)

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
		newBlockTopic, err := ps.Join("NewBlock")
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
		currentBlock := block.NewBlock(trxPool, []byte("GenesisBlock"), 1, minerKeyPair)
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
