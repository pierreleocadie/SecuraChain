package visualisation

type VisualisationData struct {
	Sender                   string   `json:"sender"`
	ConnectedPeers           []string `json:"connectedPeers"`
	TopicsList               []string `json:"topicsList"`
	ClientAnnouncement       []string `json:"clientAnnouncement"`       // List of peers subscribed to the clientAnnouncement topic viewed from the sender
	StorageNodeResponse      []string `json:"storageNodeResponse"`      // List of peers subscribed to the storageNodeResponse topic viewed from the sender
	KeepRelayConnectionAlive []string `json:"keepRelayConnectionAlive"` // List of peers subscribed to the keepRelayConnectionAlive topic viewed from the sender
}
