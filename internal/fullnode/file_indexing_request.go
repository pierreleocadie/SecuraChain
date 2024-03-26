package fullnode

import (
	"context"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// AskForIndexingRegistry sends a request for the indexing registry over the network.
func AskForMyFiles(log *ipfsLog.ZapEventLogger, ctx context.Context, askMyFiles *pubsub.Topic, recMyFiles *pubsub.Subscription) ([]byte, string, error) {
	log.Debugln("Requesting for my files from the network")
	if err := askMyFiles.Publish(ctx, []byte("I need to know what files I uploaded")); err != nil {
		log.Errorln("Error publishing my files request : ", err)
		return nil, "", err
	}

	indexingRegistryB := make(chan []byte)
	senderID := make(chan string)
	go func() {
		for {
			msg, err := recMyFiles.Next(ctx)
			if err != nil {
				log.Errorln("Error receiving message from network: ", err)
				break
			}
			if msg != nil {
				log.Debugln("My files received from : ", senderID)
				log.Debugln("My files received : ", string(msg.Data))
				indexingRegistryB <- msg.Data
				senderID <- msg.GetFrom().String()
				break
			}
		}
	}()

	return <-indexingRegistryB, <-senderID, nil
}
