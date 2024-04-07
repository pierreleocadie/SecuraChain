package visualisation

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	ipfsLog "github.com/ipfs/go-log/v2"
)

// WebSocketMessage represents a message sent over a WebSocket connection.
type WebSocketMessage struct {
	Type string      `json:"type"` // Type specifies the type of the message.
	Data interface{} `json:"data"` // Data contains the payload of the message.
}

// SendDataToClients sends the given WebSocketMessage to all connected clients.
// It takes a clientsMutex to synchronize access to the clients map,
// a clients map that stores the connected clients,
// a message of type WebSocketMessage to be sent,
// and a log of type *ipfsLog.ZapEventLogger for logging errors.
// It iterates over the clients map and sends the message to each client using WriteJSON.
// If there is an error while sending the message, the client is closed and removed from the clients map.
func SendDataToClients(clientsMutex *sync.Mutex, clients map[*websocket.Conn]bool,
	message WebSocketMessage, log *ipfsLog.ZapEventLogger) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	for client := range clients {
		err := client.WriteJSON(message)
		if err != nil {
			log.Errorf("error: %v", err)
			err := client.Close()
			if err != nil {
				return
			}
			delete(clients, client)
		}
	}
}

// CreateHandler returns an http.HandlerFunc that upgrades the HTTP connection to a WebSocket connection
// and handles incoming WebSocket connections. It takes an `upgrader` of type `websocket.Upgrader` to upgrade
// the connection, a slice of `Data` for storing data, a `clientsMutex` of type `*sync.Mutex` to synchronize
// access to the `clients` map, a `clients` map of type `map[*websocket.Conn]bool` to store connected clients,
// and a `log` of type `*ipfsLog.ZapEventLogger` for logging events.
//
// The returned http.HandlerFunc upgrades the connection to a WebSocket connection and calls the `handleConnections`
// function to handle incoming WebSocket connections.
func CreateHandler(upgrader websocket.Upgrader, data *[]Data, clientsMutex *sync.Mutex,
	clients map[*websocket.Conn]bool, log *ipfsLog.ZapEventLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Vous pouvez maintenant utiliser someParam dans votre gestionnaire
		handleConnections(w, r, upgrader, *data, clientsMutex, clients, log)
	}
}

// handleConnections handles the WebSocket connections from clients.
// It upgrades the HTTP connection to a WebSocket connection, adds the client to the list of connected clients,
// sends the initial data to the client, and listens for incoming messages from the client.
// When a client disconnects, it removes the client from the list of connected clients.
//
// Parameters:
// - w: http.ResponseWriter - the response writer for the HTTP request
// - r: *http.Request - the HTTP request
// - upgrader: websocket.Upgrader - the WebSocket upgrader
// - data: []Data - the initial data to send to the client
// - clientsMutex: *sync.Mutex - the mutex for synchronizing access to the clients map
// - clients: map[*websocket.Conn]bool - the map of connected clients
// - log: *ipfsLog.ZapEventLogger - the logger for logging events
//
// Returns: None
func handleConnections(w http.ResponseWriter, r *http.Request,
	upgrader websocket.Upgrader, data []Data, clientsMutex *sync.Mutex,
	clients map[*websocket.Conn]bool, log *ipfsLog.ZapEventLogger) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorln(err)
		return
	}
	defer func(ws *websocket.Conn) {
		err := ws.Close()
		if err != nil {
			return
		}
	}(ws)
	clientsMutex.Lock()
	clients[ws] = true
	clientsMutex.Unlock()

	// Send the initial data to the client
	initDataMessage := WebSocketMessage{
		Type: "initData",
		Data: data,
	}
	SendDataToClients(clientsMutex, clients, initDataMessage, log)

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			clientsMutex.Lock()
			delete(clients, ws)
			clientsMutex.Unlock()
			break
		}
	}
}
