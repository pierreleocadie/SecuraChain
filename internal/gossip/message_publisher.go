package gossip

import (
	"context"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// MessagePublisher publie des messages
type MessagePublisher struct {
	ctx   context.Context
	topic *pubsub.Topic
	log   *ipfsLog.ZapEventLogger
}

// NewMessagePublisher crée une nouvelle instance de MessagePublisher
func NewMessagePublisher(ctx context.Context, topic *pubsub.Topic, log *ipfsLog.ZapEventLogger) *MessagePublisher {
	return &MessagePublisher{
		ctx:   ctx,
		topic: topic,
		log:   log,
	}
}

// Publish publie un message
func (mp *MessagePublisher) Publish(data []byte) {
	err := mp.topic.Publish(mp.ctx, data)
	if err != nil {
		mp.log.Errorf("Failed to publish message: %s", err)
		return
	}
	mp.log.Debugf("Message sent successfully")
}
