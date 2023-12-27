// fullNode.go

package internal

import (
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"

	//Pebble pour la BD
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

var (
	ip4tcp  string = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 0)
	ip6tcp  string = fmt.Sprintf("/ip6/::/tcp/%d", 0)
	ip4quic string = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", 0)
	ip6quic string = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", 0)
)

// FullNode represents a full node in the SecuraChain network.
type FullNode struct {
	host      host.Host
	pubsub    *pubsub.PubSub
	blockChan chan []byte // Channel to handle incoming blocks
	db *pebble.DB
}

// NewFullNode creates a new instance of a full node.
func NewFullNode(ctx context.Context) (*FullNode, error) {
	host, err := libp2p.New(
		libp2p.ListenAddrStrings(ip4tcp, ip6tcp, ip4quic, ip6quic),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.DefaultSecurity,
		libp2p.DefaultMuxers,
	)
	if err != nil {
		return nil, err
	}

	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, err
	}

	db, err := pebble.Open("CHEMIN REEL DE LA BD PEBBLE", &pebble.Options{FS: vfs.Default})
	if err != nil {
	   return nil, err
	}

	return &FullNode{
		host:      host,
		pubsub:    ps,
		blockChan: make(chan []byte),
		db: db,
	}, nil
}

// SubscribeToMinerTopic subscribes to the miner's block topic.
func (fn *FullNode) SubscribeToMinerTopic(topicName string) error {
	topic, err := fn.pubsub.Join(topicName)
	if err != nil {
		return err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, err := sub.Next(context.Background())
			if err != nil {
				log.Println("Failed to get next block:", err)
				continue
			}

			// Process the incoming block (you might want to validate it here)
			block := msg.Data

			// Publish the block to the full node topic
			if err := fn.PublishToFullNodeTopic(block); err != nil {
				log.Println("Failed to publish block to full node:", err)
			}
		}
	}()

	// Handle incoming blocks in a separate goroutine
go func() {
	for {
	   msg, err := subBlock.Next(ctx)
	   if err != nil {
		  log.Println("Failed to get next block:", err)
		  continue
	   }
 
	   // Process the incoming block (you might want to validate it here)
	   block := msg.Data
 
	   // Store the block in the Pebble database
	   if err := fn.StoreBlockInDB(block); err != nil {
		  log.Println("Failed to store block in database:", err)
	   }
 
	   // Publish the block to the full node topic
	   if err := fn.PublishToFullNodeTopic(block); err != nil {
		  log.Println("Failed to publish block to full node:", err)
	   }
	}
 }()

	return nil
}

// PublishToFullNodeTopic publishes the given block to the full node topic.
func (fn *FullNode) PublishToFullNodeTopic(block []byte) error {
	topic, err := fn.pubsub.Join("FullNodeTopic") // Change "FullNodeTopic" to the actual topic name for full nodes
	if err != nil {
		return err
	}

	return topic.Publish(context.Background(), block)
}

// Main function to run the full node.
func (fn *FullNode) Run(ctx context.Context) {
	// Handle connection events (optional)
	subNet, err := fn.host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
	if err != nil {
		log.Println("Failed to subscribe to EvtPeerConnectednessChanged:", err)
	}
	defer subNet.Close()

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

func (fn *FullNode) StoreBlockInDB(block []byte) error {
	return fn.db.Set([]byte("Notre clÃ© en bd"+time.Now().String()), block, pebble.Sync)
}

defer func() {
	if err := fn.db.Close(); err != nil {
	   log.Println("Failed to close the database:", err)
	}
 }()