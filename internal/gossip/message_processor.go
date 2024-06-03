package gossip

import (
	"context"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// MessageProcessor traite les messages entrants
type MessageProcessor struct {
	ctx          context.Context
	subscription *pubsub.Subscription
	handler      MessageHandler
	log          *ipfsLog.ZapEventLogger
}

// NewMessageProcessor crée une nouvelle instance de MessageProcessor
func NewMessageProcessor(ctx context.Context, subscription *pubsub.Subscription, handler MessageHandler, log *ipfsLog.ZapEventLogger) *MessageProcessor {
	return &MessageProcessor{
		ctx:          ctx,
		subscription: subscription,
		handler:      handler,
		log:          log,
	}
}

// Start démarre le traitement des messages
func (mp *MessageProcessor) Start() {
	go func() {
		for {
			msg, err := mp.subscription.Next(mp.ctx)
			if err != nil {
				mp.log.Errorf("Failed to get next message: %s", err)
				continue
			}
			mp.handler.HandleMessage(msg)
		}
	}()
}
