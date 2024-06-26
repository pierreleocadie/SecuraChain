package node

import pubsub "github.com/libp2p/go-libp2p-pubsub"

type PubSubHub struct {
	ClientAnnouncementTopic       *pubsub.Topic
	ClientAnnouncementSub         *pubsub.Subscription
	StorageNodeResponseTopic      *pubsub.Topic
	StorageNodeResponseSub        *pubsub.Subscription
	KeepRelayConnectionAliveTopic *pubsub.Topic
	KeepRelayConnectionAliveSub   *pubsub.Subscription
	BlockAnnouncementTopic        *pubsub.Topic
	BlockAnnouncementSub          *pubsub.Subscription
	AskingBlockchainTopic         *pubsub.Topic
	AskingBlockchainSub           *pubsub.Subscription
	ReceiveBlockchainTopic        *pubsub.Topic
	ReceiveBlockchainSub          *pubsub.Subscription
	AskMyFilesTopic               *pubsub.Topic
	AskMyFilesSub                 *pubsub.Subscription
	SendMyFilesTopic              *pubsub.Topic
	SendMyFilesSub                *pubsub.Subscription
	NetworkVisualisationTopic     *pubsub.Topic
	NetworkVisualisationSub       *pubsub.Subscription
}
