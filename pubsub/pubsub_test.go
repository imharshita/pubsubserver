package pubsub

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHandleSubscribe(t *testing.T) {
	ps := NewPubSubServer()

	_, err := ps.newMockConn()
	if err != nil {
		log.Fatalf("cannot make websocket connection: %v", err)
	}

	// Check that the connection was added to the server's list of connections
	if len(ps.conns) != 1 {
		t.Errorf("Expected 1 connection, but got %d", len(ps.conns))
	}
}

func TestHandlePublish(t *testing.T) {
	ps := NewPubSubServer()

	conn1, err := ps.newMockConn()
	if err != nil {
		log.Fatalf("cannot make websocket connection: %v", err)
	}
	conn2, err := ps.newMockConn()
	if err != nil {
		log.Fatalf("cannot make websocket connection: %v", err)
	}

	// Create a message to publish
	testMsg := "test message"
	msg := message{Action: "publish", Message: testMsg}

	// Encode the message as JSON and send it as the request body
	body, _ := json.Marshal(msg)
	req, err := http.NewRequest("POST", "/publish", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}

	// Create a response recorder to capture the response
	rr := httptest.NewRecorder()
	ps.handlePublish(rr, req)

	// Check that the response status code is 200 OK
	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d", http.StatusOK, rr.Code)
	}

	// Check that the message was sent to the connections
	var receivedMsg string
	conn1.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := conn1.ReadJSON(&receivedMsg); err != nil {
		t.Fatal(err)
	}
	if receivedMsg != testMsg {
		t.Errorf("Expected message '%s' but got '%s'", testMsg, receivedMsg)
	}

	conn2.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := conn2.ReadJSON(&receivedMsg); err != nil {
		t.Fatal(err)
	}
	if receivedMsg != testMsg {
		t.Errorf("Expected message '%s' but got '%s'", testMsg, receivedMsg)
	}
}

// newMockConn create a new websocket connection and uses handleSubscribe handler upgrades the connection and which adds the new WebSocket
// connection to the list of connections.
func (ps *PubSubServer) newMockConn() (*websocket.Conn, error) {
	srv := httptest.NewServer(http.HandlerFunc(ps.handleSubscribe))
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	return conn, err
}
