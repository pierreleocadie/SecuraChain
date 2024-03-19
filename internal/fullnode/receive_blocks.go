package fullnode

import (
	"context"
	"fmt"

	ipfsLog "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

// ReceiveBlock receives a block announcement message and deserializes it into a block.
func ReceiveBlock(log *ipfsLog.ZapEventLogger, ctx context.Context, subBlockAnnouncement *pubsub.Subscription) (*block.Block, error) {
	msg, err := subBlockAnnouncement.Next(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting block announcement message: %s", err)
	}

	log.Debugln("Received block announcement message from ", msg.GetFrom().String())
	log.Debugln("Received block: ", msg.Data)

	// Deserialize the block announcement
	b, err := block.DeserializeBlock(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("error deserializing block announcement: %s", err)
	}
	return b, nil
}
