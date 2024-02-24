package fullnode

import (
	"context"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// PublishBlockToNetwork publishes a block to the network. (for minors and indexing and searching nodes and storage nodes)
func PublishBlockToNetwork(ctx context.Context, blockk *block.Block, fullNodeAnnouncementTopic *pubsub.Topic) (bool, error) {
	// Serialize the block
	blockBytes, err := blockk.Serialize()
	if err != nil {
		return false, fmt.Errorf("Error serializing block : %s", err)
	}

	if err = fullNodeAnnouncementTopic.Publish(ctx, blockBytes); err != nil {
		return false, fmt.Errorf("Error publishing block to the network : %s", err)
	}

	return true, nil
}
