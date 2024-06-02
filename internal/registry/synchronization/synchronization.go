package synchronization

import (
	"context"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/pierreleocadie/SecuraChain/internal/config"
	"github.com/pierreleocadie/SecuraChain/internal/registry"
)

// AskForBlockchainRegistry sends a request for the blockchain registry over the network.
func AskForBlockchainRegistry(log *ipfsLog.ZapEventLogger, ctx context.Context, askBlockchain *pubsub.Topic, recBlockchain *pubsub.Subscription) ([]byte, string, error) {
	log.Debugln("Requesting blockchain from the network")
	if err := askBlockchain.Publish(ctx, []byte("I need to synchronize. Who can help me ?")); err != nil {
		log.Errorln("Error publishing blockchain request : ", err)
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
				log.Debugln("Registry received from : ", senderID)
				log.Debugln("Registry received : ", string(msg.Data))
				registryBytes <- msg.Data
				senderID <- msg.GetFrom().String()
				break
			}
		}
	}()

	return <-registryBytes, <-senderID, nil
}

// SendBlocksRegistryToNetwork sends the registry of the blockchain to the network.
func SendBlocksRegistryToNetwork(log *ipfsLog.ZapEventLogger, ctx context.Context, config *config.Config, network *pubsub.Topic) bool {
	// Get the registry of the blockchain
	r, err := registry.LoadRegistryFile[registry.BlockRegistry](log, config, config.BlockRegistryPath)
	if err != nil {
		log.Errorln("Error loading the registry of the blockchain : ", err)
		return false
	}

	registryBytes, err := registry.SerializeRegistry(log, r)
	if err != nil {
		log.Errorln("Error serializing the registry of the blockchain : ", err)
		return false
	}
	if err := network.Publish(ctx, registryBytes); err != nil {
		log.Errorln("Error publishing the registry of the blockchain : ", err)
		return false
	}

	log.Debugln("Blocks registry sent to the network")
	return true
}
