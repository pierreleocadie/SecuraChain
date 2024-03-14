package fullnode

import (
	"context"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pierreleocadie/SecuraChain/internal/core/block"
)

func ReceiveBlock(ctx context.Context, subBlockAnnouncement *pubsub.Subscription) (*block.Block, error) {
	msg, err := subBlockAnnouncement.Next(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting block announcement message : %s", err)
	}

	// fmt.Println("Received block announcement message from ", msg.GetFrom().String())
	// fmt.Println("Received block : ", msg.Data)

	// Deserialize the block announcement
	blockAnnounced, err := block.DeserializeBlock(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("error deserializing block announcement : %s", err)
	}
	return blockAnnounced, nil
}
