package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// ReloadType classifies what kind of reload to perform.
type ReloadType string

const (
	ReloadFull    ReloadType = "reload"
	ReloadCSS     ReloadType = "css"
	ReloadError   ReloadType = "error"
	ReloadWarning ReloadType = "warning"
)

// ReloadMessage is sent to browsers over WebSocket.
type ReloadMessage struct {
	Type  ReloadType `json:"type"`
	Path  string     `json:"path,omitempty"`
	Error string     `json:"error,omitempty"`
	File  string     `json:"file,omitempty"`
	Line  int        `json:"line,omitempty"`
	Col   int        `json:"col,omitempty"`
	Frame string     `json:"frame,omitempty"`
}

// Hub manages WebSocket client connections and broadcasts reload messages.
type Hub struct {
	clients  map[*websocket.Conn]bool
	mu       sync.Mutex
	upgrader websocket.Upgrader
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// HandleWS upgrades an HTTP connection to WebSocket and registers the client.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
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

// Broadcast sends a reload message to all connected WebSocket clients.
func (h *Hub) Broadcast(msg ReloadMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal reload message: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

// ClientCount returns the number of connected WebSocket clients.
func (h *Hub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}
