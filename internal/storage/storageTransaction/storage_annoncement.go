package storageTransaction

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/ipfs/kubo/core"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/core/transaction"
	ecdsaSC "github.com/pierreleocadie/SecuraChain/pkg/ecdsa"
)

var channelStorageNodeReponse string = "StorageNodeResponse"

func StorageAnnoncement(ctx context.Context, ps *pubsub.PubSub, ipfsNode *core.IpfsNode, annoncement *transaction.ClientAnnouncement) {

	/*
	* GENERATE ECDSA KEY PAIR FOR NODE IDENTITY
	 */
	// Generate a pair of ecdsa keys
	keyPair, err := ecdsaSC.NewECDSAKeyPair()
	if err != nil {
		log.Println("Failed to generate ecdsa key pair:", err)
	}

	// --------------- Créer la réponse --------------------
	response := transaction.NewStorageNodeResponse(keyPair, ipfsNode.Identity, "http://localhost:8081/data", annoncement)

	// ----------------- Serialize avant d'envoyer  ---------------
	serializeData, err := response.Serialize()
	if err != nil {
		log.Println("Error when serialize data :", err)
	}

	// Join the topic
	topicStorageNodeAnnoncement, err := ps.Join(channelStorageNodeReponse)
	if err != nil {
		log.Println("Failed to join topic:", err)
	}

	go func() {
		for {
			time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
			// Publish the transaction
			if err := topicStorageNodeAnnoncement.Publish(ctx, serializeData); err != nil {
				log.Println("Failed to publish transaction:", err)
			}
		}
	}()

}
