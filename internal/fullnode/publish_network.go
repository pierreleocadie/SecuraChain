package fullnode

import (
	"context"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// PublishBlockToNetwork publishes a block to the network.
func PublishBlockToNetwork(ctx context.Context, b *block.Block, blockAnnouncementTopic *pubsub.Topic) (bool, error) {
	// Serialize the block
	blockBytes, err := b.Serialize()
	if err != nil {
		return false, fmt.Errorf("error serializing block: %s", err)
	}

	if err = blockAnnouncementTopic.Publish(ctx, blockBytes); err != nil {
		return false, fmt.Errorf("error publishing block to the network: %s", err)
	}

	return true, nil
}
