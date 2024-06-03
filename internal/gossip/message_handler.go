package gossip

import pubsub "github.com/libp2p/go-libp2p-pubsub"

// MessageHandler interface pour le traitement des messages
type MessageHandler interface {
	HandleMessage(msg *pubsub.Message)
}
