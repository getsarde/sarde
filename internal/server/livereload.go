package server

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/getsarde/sarde/internal/devlog"
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

// WarningItem is a single structured warning for the browser overlay.
type WarningItem struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// ReloadMessage is sent to browsers over WebSocket.
type ReloadMessage struct {
	Type      ReloadType    `json:"type"`
	Path      string        `json:"path,omitempty"`
	Error     string        `json:"error,omitempty"`
	File      string        `json:"file,omitempty"`
	Line      int           `json:"line,omitempty"`
	Col       int           `json:"col,omitempty"`
	Frame     string        `json:"frame,omitempty"`
	ChangedAt int64         `json:"changedAt,omitempty"` // Unix millis when the file change was first detected
	Warnings  []WarningItem `json:"warnings,omitempty"`
}

// Hub manages WebSocket client connections and broadcasts reload messages.
type Hub struct {
	clients      map[*websocket.Conn]bool
	mu           sync.Mutex
	upgrader     websocket.Upgrader
	pendingError  *ReloadMessage
	pendingReload *ReloadMessage
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

// SetPendingError stores a build error to replay to newly connecting clients.
func (h *Hub) SetPendingError(msg *ReloadMessage) {
	h.mu.Lock()
	h.pendingError = msg
	h.mu.Unlock()
}

// ClearPendingError removes the stored build error after a successful rebuild.
func (h *Hub) ClearPendingError() {
	h.mu.Lock()
	h.pendingError = nil
	h.mu.Unlock()
}

// SetPendingReload stores a successful reload message to replay to newly
// connecting clients that missed the original broadcast.
func (h *Hub) SetPendingReload(msg *ReloadMessage) {
	h.mu.Lock()
	h.pendingReload = msg
	h.mu.Unlock()
}

// ClearPendingReload removes the stored reload message.
func (h *Hub) ClearPendingReload() {
	h.mu.Lock()
	h.pendingReload = nil
	h.mu.Unlock()
}

// HandleWS upgrades an HTTP connection to WebSocket and registers the client.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		devlog.Error("ws", "WebSocket upgrade error: %v", err)
		return
	}

	// Replay any pending build error inside the locked section: all writes to
	// a conn must happen under h.mu (Broadcast writes under it too), since
	// gorilla/websocket connections do not support concurrent writers.
	h.mu.Lock()
	h.clients[conn] = true
	if h.pendingError != nil {
		if data, err := json.Marshal(h.pendingError); err == nil {
			conn.WriteMessage(websocket.TextMessage, data)
		}
	} else if h.pendingReload != nil {
		if data, err := json.Marshal(h.pendingReload); err == nil {
			conn.WriteMessage(websocket.TextMessage, data)
		}
		h.pendingReload = nil
	}
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
// Returns the number of clients that received the message.
func (h *Hub) Broadcast(msg ReloadMessage) int {
	data, err := json.Marshal(msg)
	if err != nil {
		devlog.Error("ws", "Failed to marshal reload message: %v", err)
		return 0
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	sent := 0
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			conn.Close()
			delete(h.clients, conn)
		} else {
			sent++
		}
	}
	return sent
}

// ClientCount returns the number of connected WebSocket clients.
func (h *Hub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}
