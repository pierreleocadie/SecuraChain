package storageTransaction

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/ipfs/kubo/core"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	ecdsaSC "github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

// var (
// // transactionTopicNameFlag *string = flag.String("StorageNodeResponse")
// // ip4tcp                   string  = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 0)
// // ip6tcp                   string  = fmt.Sprintf("/ip6/::/tcp/%d", 0)
// // ip4quic                  string  = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", 0)
// // ip6quic                  string  = fmt.Sprintf("/ip6/::/udp/%d/quic-v1", 0)
// )

func StorageAnnoncement(ctx context.Context, ipfsNode *core.IpfsNode, data []byte, annoncement *transaction.ClientAnnouncement) {

	var transactionTopicNameFlag *string = flag.String("StorageNodeResponse", "SecuraChain", "Unique string to identify group of nodes. Share this with your friends to let them connect with you")

	/*
	* GENERATE ECDSA KEY PAIR FOR NODE IDENTITY
	 */
	// Generate a pair of ecdsa keys
	keyPair, err := ecdsaSC.NewECDSAKeyPair()
	if err != nil {
		log.Println("Failed to generate ecdsa key pair:", err)
	}

	host := ipfsNode.PeerHost

	// --------------- Créer la réponse --------------------
	response := transaction.NewStorageNodeResponse(keyPair, ipfsNode.Identity, "apiendpoint", annoncement)

	// ----------------- Serialize avant d'envoyer  ---------------
	serializeData, err := response.Serialize()
	if err != nil {
		log.Println("Error when serialize data :", err)
	}

	/*
	* PUBLISH TO StorageNodeResponse
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

	// Every X seconds, publish a new transaction - random interval between 1 and 10 seconds
	go func() {
		for {
			time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
			// Publish the transaction
			if err := topicTrx.Publish(ctx, serializeData); err != nil {
				log.Println("Failed to publish transaction:", err)
			}
		}
	}()

}
