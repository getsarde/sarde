package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHub_BroadcastNoClients(t *testing.T) {
	hub := NewHub()
	// Should not panic with no clients.
	hub.Broadcast(ReloadMessage{Type: ReloadFull})
}

func TestHub_ClientCount(t *testing.T) {
	hub := NewHub()
	if got := hub.ClientCount(); got != 0 {
		t.Errorf("ClientCount() = %d, want 0", got)
	}
}

func TestHub_ConnectAndBroadcast(t *testing.T) {
	hub := NewHub()

	// Create test server with WebSocket endpoint.
	srv := httptest.NewServer(http.HandlerFunc(hub.HandleWS))
	defer srv.Close()

	// Connect WebSocket client.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	// Wait for client registration.
	time.Sleep(50 * time.Millisecond)

	if got := hub.ClientCount(); got != 1 {
		t.Fatalf("ClientCount() = %d, want 1", got)
	}

	// Broadcast a message.
	sent := ReloadMessage{Type: ReloadFull, Path: "/docs/intro/"}
	hub.Broadcast(sent)

	// Read the message on the client.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	var received ReloadMessage
	if err := json.Unmarshal(data, &received); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if received.Type != ReloadFull {
		t.Errorf("Type = %q, want %q", received.Type, ReloadFull)
	}
	if received.Path != "/docs/intro/" {
		t.Errorf("Path = %q, want %q", received.Path, "/docs/intro/")
	}
}

// A newly connecting client must receive the pending build error.
func TestHub_PendingErrorReplay(t *testing.T) {
	hub := NewHub()
	hub.SetPendingError(&ReloadMessage{Type: ReloadError, Error: "boom"})

	srv := httptest.NewServer(http.HandlerFunc(hub.HandleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	var received ReloadMessage
	if err := json.Unmarshal(data, &received); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if received.Type != ReloadError || received.Error != "boom" {
		t.Errorf("received %+v, want pending error replay", received)
	}
}

// Clients connecting (each triggering a pending-error replay write) while the
// hub broadcasts must not write to the same conn concurrently. Run with -race;
// before the fix the replay write happened outside h.mu and raced Broadcast.
func TestHub_ConcurrentConnectAndBroadcast(t *testing.T) {
	hub := NewHub()
	hub.SetPendingError(&ReloadMessage{Type: ReloadError, Error: "pending"})

	srv := httptest.NewServer(http.HandlerFunc(hub.HandleWS))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			hub.Broadcast(ReloadMessage{Type: ReloadFull, Path: "/"})
		}
	}()

	var conns []*websocket.Conn
	for i := 0; i < 8; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("WebSocket dial %d failed: %v", i, err)
		}
		conns = append(conns, conn)
		// Drain a message so the server-side writes interleave with reads.
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Fatalf("ReadMessage %d failed: %v", i, err)
		}
	}
	<-done

	for _, c := range conns {
		c.Close()
	}
}

func TestHub_BroadcastError(t *testing.T) {
	hub := NewHub()

	srv := httptest.NewServer(http.HandlerFunc(hub.HandleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// Broadcast an error message.
	sent := ReloadMessage{
		Type:  ReloadError,
		Error: "template parse error",
		File:  "layouts/blog/single.html",
		Line:  23,
		Col:   12,
	}
	hub.Broadcast(sent)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	var received ReloadMessage
	if err := json.Unmarshal(data, &received); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if received.Type != ReloadError {
		t.Errorf("Type = %q, want %q", received.Type, ReloadError)
	}
	if received.File != "layouts/blog/single.html" {
		t.Errorf("File = %q, want %q", received.File, "layouts/blog/single.html")
	}
	if received.Line != 23 {
		t.Errorf("Line = %d, want 23", received.Line)
	}
}
