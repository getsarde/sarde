package project

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestEventHub_BroadcastNoClients(t *testing.T) {
	hub := NewEventHub()
	// Should not panic.
	hub.Broadcast(Event{Type: "test"})
}

func TestEventHub_ConnectAndBroadcast(t *testing.T) {
	hub := NewEventHub()

	srv := httptest.NewServer(http.HandlerFunc(hub.HandleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Fatalf("ClientCount = %d, want 1", hub.ClientCount())
	}

	hub.Broadcast(Event{Type: "file:created", Data: map[string]any{"path": "blog/hello.md"}})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	var received Event
	if err := json.Unmarshal(data, &received); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if received.Type != "file:created" {
		t.Errorf("Type = %q, want file:created", received.Type)
	}
}
