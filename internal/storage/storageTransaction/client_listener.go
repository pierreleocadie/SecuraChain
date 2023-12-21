package storageTransaction

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

const RefreshInterval = 10 * time.Second

var (
	ignoredPeers             map[peer.ID]bool = make(map[peer.ID]bool)
	channelClientAnnoncement string           = "ClientAnnouncement"
	transactionTopicNameFlag *string          = flag.String("transactionTopicName", "NewTransaction", "ClientAnnouncement")
	ip4tcp                   string           = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 0)
	ip6tcp                   string           = fmt.Sprintf("/ip6/::/tcp/%d", 0)
	ip4quic                  string           = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", 0)
	ip6quic                  string           = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", 0)
)

func SubscribeToClientChannel(ctx context.Context, ps *pubsub.PubSub) {

	// host := ipfsNode.PeerHost

	/*
	* SUBSCRIBE TO ClientAnnouncement
	 */
	// // Create a new PubSub service using the GossipSub router
	// ps, err := pubsub.NewGossipSub(ctx, host)
	// if err != nil {
	// 	log.Println("Failed to create new PubSub service:", err)
	// }

	// Join the topic client channel
	topicClient, err := ps.Join(channelClientAnnoncement)
	if err != nil {
		log.Println("Failed to join topic:", err)
	}
	subClient, err := topicClient.Subscribe()
	if err != nil {
		log.Println("Failed to subscribe to topic client:", err)
	}

	go func() {
		for {
			msg, err := subClient.Next(ctx)
			if err != nil {
				log.Println("Failed to read next message:", err)
				continue
			}
			fmt.Printf("Received message: %s\n", string(msg.Data))
		}
	}()

}
