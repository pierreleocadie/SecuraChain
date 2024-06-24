package blockregistry

import (
	"context"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// AskForBlockchainRegistry sends a request for the blockchain registry over the network.
func AskForBlockchainRegistry(log *ipfsLog.ZapEventLogger, ctx context.Context, askBlockchain *pubsub.Topic, recBlockchain *pubsub.Subscription) ([]byte, string, error) {
	log.Debugln("Requesting block registry from the network")
	if err := askBlockchain.Publish(ctx, []byte("I need to synchronize. Who can help me ?")); err != nil {
		log.Errorln("Error publishing block registry request : ", err)
		return nil, "", err
	}

	registryBytes := make(chan []byte)
	senderID := make(chan string)
	go func() {
		for {
			msg, err := recBlockchain.Next(ctx)
			if err != nil {
				log.Errorln("Error receiving message from network: ", err)
				break
			}
			if msg != nil {
				log.Debugln("Block registry received from : ", msg.GetFrom().String())
				log.Debugln("Block registry received : ", string(msg.Data))
				registryBytes <- msg.Data
				senderID <- msg.GetFrom().String()
				break
			}
		}
	}()

	return <-registryBytes, <-senderID, nil
}
