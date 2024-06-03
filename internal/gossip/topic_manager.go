package gossip

import (
	"context"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/pierreleocadie/SecuraChain/internal/config"
)

// TopicManager gère les topics de PubSub
type TopicManager struct {
	ctx  context.Context
	ps   *pubsub.PubSub
	host host.Host
	cfg  *config.Config
	log  *ipfsLog.ZapEventLogger
}

// NewTopicManager crée une nouvelle instance de TopicManager
func NewTopicManager(ctx context.Context, ps *pubsub.PubSub, host host.Host, cfg *config.Config, log *ipfsLog.ZapEventLogger) *TopicManager {
	return &TopicManager{
		ctx:  ctx,
		ps:   ps,
		host: host,
		cfg:  cfg,
		log:  log,
	}
}

// JoinAndSubscribeTopic rejoint et s'abonne à un topic
func (tm *TopicManager) JoinAndSubscribeTopic(topicName string) (*pubsub.Topic, *pubsub.Subscription, error) {
	topic, err := tm.ps.Join(topicName)
	if err != nil {
		tm.log.Warnf("Failed to join topic %s: %s", topicName, err)
		return nil, nil, err
	}

	subscription, err := topic.Subscribe()
	if err != nil {
		tm.log.Warnf("Failed to subscribe to topic %s: %s", topicName, err)
		return nil, nil, err
	}

	return topic, subscription, nil
}
