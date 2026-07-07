package server

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/getsarde/sarde/internal/devlog"
	"github.com/gorilla/websocket"
)

// sameOriginWS accepts the upgrade only when the page opening the socket was
// served by this dev server (the injected live-reload client always connects
// same-origin via location.host). Empty Origin means a non-browser client and
// is allowed; a foreign web page's Origin will not match the Host it dialed.
func sameOriginWS(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	// localhost and 127.0.0.1 are the same listener spelled two ways.
	return loopbackHostPort(u.Host) != "" && loopbackHostPort(u.Host) == loopbackHostPort(r.Host)
}

// loopbackHostPort canonicalizes "localhost:p" / "127.0.0.1:p" / "[::1]:p" to
// "loopback:p"; returns "" for non-loopback hosts.
func loopbackHostPort(hostport string) string {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		// No port (e.g. bare "localhost" on default-port URLs).
		host, port = hostport, ""
	}
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return "loopback:" + port
	}
	return ""
}

// ReloadType classifies what kind of reload to perform.
type ReloadType string

const (
	ReloadFull    ReloadType = "reload"
	ReloadCSS     ReloadType = "css"
	ReloadError   ReloadType = "error"
	ReloadWarning ReloadType = "warning"
	// ReloadSync announces the server's latest successful build ID to a client
	// that just connected. The client compares it against the build ID embedded
	// in its page and reloads only if the page predates the build, so stale
	// tabs (reconnects, server restarts, missed broadcasts) catch up exactly
	// once and fresh tabs never reload spuriously.
	ReloadSync ReloadType = "sync"
)

// writeTimeout bounds each WebSocket write so one stalled client cannot hold
// h.mu (and with it all broadcasts and new connections) indefinitely.
const writeTimeout = 5 * time.Second

var (
	pingInterval = 30 * time.Second
	pongWait     = 35 * time.Second
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
	BuildID   int64         `json:"buildId,omitempty"`   // monotonic ID of the successful build this message refers to
	Warnings  []WarningItem `json:"warnings,omitempty"`
}

// Hub manages WebSocket client connections and broadcasts reload messages.
type Hub struct {
	clients      map[*websocket.Conn]bool
	mu           sync.Mutex
	upgrader     websocket.Upgrader
	pendingError *ReloadMessage
	buildID      int64 // latest successful build ID; seeded with the server start time so it survives restarts
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: sameOriginWS,
		},
		// Seeding with wall-clock time (rather than 0) makes IDs comparable
		// across server restarts: a page served by a previous server instance
		// always predates this instance's first build.
		buildID: time.Now().UnixMilli(),
	}
}

// BumpBuildID records a new successful build and returns its ID. IDs are
// millisecond timestamps forced monotonic, so two builds completing within
// the same millisecond (or a clock step backwards) cannot collide.
func (h *Hub) BumpBuildID() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	id := time.Now().UnixMilli()
	if id <= h.buildID {
		id = h.buildID + 1
	}
	h.buildID = id
	return id
}

// BuildID returns the ID of the latest successful build.
func (h *Hub) BuildID() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.buildID
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

// writeLocked writes a message to conn with a bounded deadline. Callers must
// hold h.mu: all writes to a conn happen under it, since gorilla/websocket
// connections do not support concurrent writers.
func (h *Hub) writeLocked(conn *websocket.Conn, data []byte) error {
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return conn.WriteMessage(websocket.TextMessage, data)
}

// HandleWS upgrades an HTTP connection to WebSocket and registers the client.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		devlog.Error("ws", "WebSocket upgrade error: %v", err)
		return
	}

	// Replay a pending build error, or announce the latest build ID so a
	// client whose page predates it can catch up (see ReloadSync).
	h.mu.Lock()
	h.clients[conn] = true
	if h.pendingError != nil {
		if data, err := json.Marshal(h.pendingError); err == nil {
			h.writeLocked(conn, data)
		}
	} else if data, err := json.Marshal(ReloadMessage{Type: ReloadSync, BuildID: h.buildID}); err == nil {
		h.writeLocked(conn, data)
	}
	h.mu.Unlock()

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeTimeout)); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Read pump — blocks until client disconnects or read deadline fires.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	close(done)

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
		if err := h.writeLocked(conn, data); err != nil {
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
