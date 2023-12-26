package storagetransaction

import (
	"context"
	"fmt"
	"log"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
)

const RefreshInterval = 10 * time.Second

var clientAnnouncementStringFlag = fmt.Sprintln("ClientAnnouncement")

func SubscribeToClientChannel(ctx context.Context, ps *pubsub.PubSub, announceChan chan *transaction.ClientAnnouncement) {
	// Join the topic client channel
	topicClient, err := ps.Join(clientAnnouncementStringFlag)
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
			transac, err := transaction.DeserializeClientAnnouncement(msg.Data)
			if err != nil {
				log.Printf("Error on desarializing Client announcement %s", err)
				continue
			}
			announceChan <- transac // Envoyez l'annonce sur le canal
		}
	}()
}
