package visualisation

type Data struct {
	PeerID                   string   `json:"peerID"`
	NodeType                 string   `json:"nodeType"`
	ConnectedPeers           []string `json:"connectedPeers"` // List of peers connected (peers ID) to this peer
	TopicsList               []string `json:"topicsList"`
	KeepRelayConnectionAlive []string `json:"keepRelayConnectionAlive"` // List of peers subscribed to KeepRelayConnectionAlive which are connected to this peer
	BlockAnnouncement        []string `json:"blockAnnouncement"`        // List of peers subscribed to BlockAnnouncement which are connected to this peer
	AskingBlockchain         []string `json:"askingBlockchain"`         // List of peers subscribed to AskingBlockchain which are connected to this peer
	ReceiveBlockchain        []string `json:"receiveBlockchain"`        // List of peers subscribed to ReceiveBlockchain which are connected to this peer
	ClientAnnouncement       []string `json:"clientAnnouncement"`       // List of peers subscribed to ClientAnnouncement which are connected to this peer
	StorageNodeResponse      []string `json:"storageNodeResponse"`      // List of peers subscribed to StorageNodeResponse which are connected to this peer
	FullNodeAnnouncement     []string `json:"fullNodeAnnouncement"`     // List of peers subscribed to FullNodeAnnouncement which are connected to this peer
	AskMyFilesList           []string `json:"askMyFilesList"`           // List of peers subscribed to AskMyFilesList which are connected to this peer
	ReceiveMyFilesList       []string `json:"receiveMyFilesList"`       // List of peers subscribed to ReceiveMyFilesList which are connected to this peer
}
