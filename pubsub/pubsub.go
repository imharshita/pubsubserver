// Package pubsubserver implements a simple pub/sub server over WebSocket.
package pubsub

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// message represents a message sent between server and clients.
type message struct {
	Action  string `json:"action"`
	Message string `json:"message"`
}

// PubSubServer enables broadcasting to a set of subscribers.
type PubSubServer struct {
	// conns stores the set of active WebSocket connections.
	conns map[*websocket.Conn]bool
	// connsMux guards access to the conns map.
	connsMux sync.Mutex
	// router routes the various endpoints to the appropriate handler.
	router http.ServeMux
	// logf controls where logs are sent.
	// Defaults to log.Printf.
	logf func(f string, v ...interface{})
}

// NewPubSubServer constructs a new PubSubServer with the default options.
func NewPubSubServer() *PubSubServer {
	ps := &PubSubServer{
		conns: make(map[*websocket.Conn]bool),
		logf:  log.Printf,
	}

	ps.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static")
	})

	ps.router.HandleFunc("/subscribe", ps.handleSubscribe)
	ps.router.HandleFunc("/publish", ps.handlePublish)

	return ps
}

// ServeHTTP serves HTTP requests using the server's internal serveMux.
func (ps *PubSubServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ps.router.ServeHTTP(w, r)
}

// handlePublish reads the request body and then publishes
// the received message to all subscribers.
func (ps *PubSubServer) handlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var msg message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		ps.logf("Error decoding request body: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if msg.Action != "publish" {
		http.Error(w, "invalid message", http.StatusBadRequest)
		return
	}

	for conn := range ps.conns {
		if err := conn.WriteJSON(msg.Message); err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway) {
				ps.logf("Connection is closing")
			} else {
				ps.logf("Error publishing message: %v", err)
			}
			if err := ps.CloseConn(conn); err != nil {
				ps.logf("Error closing connection: %v", err)
			}
			continue
		}
	}
}

// handleSubscribe accepts the WebSocket connection and then subscribes
// it to all future messages.
func (ps *PubSubServer) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}

	// Upgrade HTTP connection to WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		ps.logf("Failed to upgrade connection: %v", err)
		return
	}

	// Add the new WebSocket connection to the list of connections
	ps.connsMux.Lock()
	ps.conns[conn] = true
	ps.connsMux.Unlock()
	ps.logf("New Client is connected, total: %v", len(ps.conns))

	var msg message
	if err := conn.ReadJSON(&msg); err != nil {
		if websocket.IsCloseError(err, websocket.CloseGoingAway) {
			ps.logf("Connection is closing")
		} else {
			ps.logf("WebSocket connection error: %v", err)
		}
		if err := ps.CloseConn(conn); err != nil {
			ps.logf("%v", err)
		}
		return
	}

	// Handle the message from the client
	switch msg.Action {
	case "subscribe":
		// Client wants to subscribe
		// Do nothing, as we've already added the client to the list of connections
	case "unsubscribe":
		if err := ps.CloseConn(conn); err != nil {
			ps.logf("%v", err)
		}
	default:
		if err := conn.WriteMessage(websocket.TextMessage, []byte("invalid message")); err != nil {
			ps.logf("%v", err)
		}
		return
	}
}

// CloseConn removes a WebSocket connection from the list of connections
// and closes the connection.
func (ps *PubSubServer) CloseConn(conn *websocket.Conn) error {
	ps.connsMux.Lock()
	defer ps.connsMux.Unlock()

	delete(ps.conns, conn)
	if err := conn.Close(); err != nil {
		return err
	}
	ps.logf("Client disconnected, total: %v", len(ps.conns))
	return nil
}
