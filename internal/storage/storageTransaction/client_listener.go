package storageTransaction

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/ipfs/kubo/core"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
)

const RefreshInterval = 10 * time.Second

var (
	ignoredPeers             map[peer.ID]bool = make(map[peer.ID]bool)
	channelClientAnnoncement *string          = flag.String("ClientAnnouncement", "SecuraChain", "Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	transactionTopicNameFlag *string          = flag.String("transactionTopicName", "NewTransaction", "ClientAnnouncement")
	ip4tcp                   string           = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 0)
	ip6tcp                   string           = fmt.Sprintf("/ip6/::/tcp/%d", 0)
	ip4quic                  string           = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", 0)
	ip6quic                  string           = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", 0)
)

func SubscribeToClientChannel(ctx context.Context, ipfsNode *core.IpfsNode) (transaction.Transaction, error) {

	host := ipfsNode.PeerHost

	/*
	* SUBSCRIBE TO ClientAnnouncement
	 */
	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Println("Failed to create new PubSub service:", err)
	}

	// Join the topic
	topicTrx, err := ps.Join(*transactionTopicNameFlag)
	if err != nil {
		log.Println("Failed to join topic:", err)
	}
	subTrx, err := topicTrx.Subscribe()
	if err != nil {
		log.Println("Failed to subscribe to topic:", err)
	}

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

	select {}

}
