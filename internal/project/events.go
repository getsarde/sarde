package project

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Event is the generic event envelope sent over WebSocket.
type Event struct {
	Type string `json:"event"`
	Data any    `json:"data,omitempty"`
}

// eventWriteTimeout bounds each WebSocket write so one stalled client cannot
// hold h.mu indefinitely.
const eventWriteTimeout = 5 * time.Second

// EventHub manages WebSocket client connections and broadcasts events.
type EventHub struct {
	clients  map[*websocket.Conn]bool
	mu       sync.Mutex
	upgrader websocket.Upgrader
}

// NewEventHub creates a new event hub.
func NewEventHub() *EventHub {
	return &EventHub{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return allowedEventOrigin(r.Header.Get("Origin"))
			},
		},
	}
}

// allowedEventOrigin permits the Tauri webview origins, loopback-hosted
// frontends, and non-browser clients (empty Origin). Arbitrary web pages are
// rejected so a browser tab cannot subscribe to project events cross-origin.
// The API server's bearer-token middleware has already vetted the request
// before the upgrade reaches this check.
func allowedEventOrigin(origin string) bool {
	if origin == "" {
		return true
	}
	switch origin {
	case "tauri://localhost", "https://tauri.localhost", "http://tauri.localhost":
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// HandleWS upgrades an HTTP connection to WebSocket and registers the client.
func (h *EventHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("EventHub WebSocket upgrade error: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	// Read pump — blocks until client disconnects.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	conn.Close()
}

// Broadcast sends an event to all connected WebSocket clients.
func (h *EventHub) Broadcast(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("EventHub marshal error: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		conn.SetWriteDeadline(time.Now().Add(eventWriteTimeout))
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *EventHub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}
