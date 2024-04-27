package visualisation

import "slices"

type PubsubMessageSignal struct {
	ID           string              `json:"id"`
	From         string              `json:"from"`
	Topic        string              `json:"topic"`
	Subscribers  []string            `json:"subscribers"`
	Connections  []PeeringConnection `json:"connections"`
	VisitedNodes []string            `json:"visitedNodes"`
}

type PeeringConnection struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

func GetPeersSubscribedToTopicForPeer(peer Data, topic string) []string {
	if slices.Contains(peer.TopicsList, topic) {
		// return the list of peers subscribed to the topic
		switch topic {
		case "KeepRelayConnectionAlive":
			return peer.KeepRelayConnectionAlive
		case "BlockAnnouncement":
			return peer.BlockAnnouncement
		case "AskingBlockchain":
			return peer.AskingBlockchain
		case "ReceiveBlockchain":
			return peer.ReceiveBlockchain
		case "ClientAnnouncement":
			return peer.ClientAnnouncement
		case "StorageNodeResponse":
			return peer.StorageNodeResponse
		case "FullNodeAnnouncement":
			return peer.FullNodeAnnouncement
		case "AskMyFilesList":
			return peer.AskMyFilesList
		case "ReceiveMyFilesList":
			return peer.ReceiveMyFilesList
		}
	}
	return []string{}
}

func GetPeersSubscribedToTopic(topic string, data []Data) ([]string, []PeeringConnection) {
	peers := []string{}
	connections := []PeeringConnection{}

	for _, node := range data {
		if slices.Contains(node.TopicsList, topic) {
			peers = append(peers, node.PeerID)
			connectedPeersSubscribedToTopic := GetPeersSubscribedToTopicForPeer(node, topic)
			for _, connectedPeer := range node.ConnectedPeers {
				if slices.Contains(connectedPeersSubscribedToTopic, connectedPeer) {
					connections = append(connections, PeeringConnection{
						Source: node.PeerID,
						Target: connectedPeer,
					})
				}
			}
		}
	}

	return peers, connections
}
