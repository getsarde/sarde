package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coderoo-dev/coderoo/internal/project"
)

// APIServer serves the IPC JSON API for the desktop app.
type APIServer struct {
	pm     *project.ProjectManager
	hub    *project.EventHub
	server *http.Server
	port   int
}

// NewAPIServer creates a new API server.
func NewAPIServer(pm *project.ProjectManager, hub *project.EventHub) *APIServer {
	return &APIServer{pm: pm, hub: hub}
}

// Start begins listening on the given port. If port is 0, an ephemeral port is assigned.
// Returns the actual port being served on.
func (s *APIServer) Start(port int) (int, error) {
	mux := http.NewServeMux()
	s.setupRoutes(mux)

	// Apply middleware: logger → recoverer → cors → mux.
	var handler http.Handler = mux
	handler = corsMiddleware(handler)
	handler = recovererMiddleware(handler)
	handler = loggerMiddleware(handler)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return 0, fmt.Errorf("listening on port %d: %w", port, err)
	}

	s.port = listener.Addr().(*net.TCPAddr).Port
	s.server = &http.Server{Handler: handler}

	go s.server.Serve(listener)
	return s.port, nil
}

// Stop gracefully shuts down the API server.
func (s *APIServer) Stop() error {
	if s.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// Port returns the port the server is listening on.
func (s *APIServer) Port() int {
	return s.port
}

func (s *APIServer) setupRoutes(mux *http.ServeMux) {
	// Project lifecycle.
	mux.HandleFunc("POST /api/project/open", s.handleProjectOpen)
	mux.HandleFunc("POST /api/project/create", s.handleProjectCreate)
	mux.HandleFunc("POST /api/project/close", s.handleProjectClose)
	mux.HandleFunc("GET /api/project/info", s.handleProjectInfo)

	// Content CRUD.
	mux.HandleFunc("GET /api/content", s.handleListContent)
	mux.HandleFunc("POST /api/content", s.handleCreateContent)
	mux.HandleFunc("POST /api/content/rename", s.handleRenameContent)
	mux.HandleFunc("GET /api/content/{path...}", s.handleReadContent)
	mux.HandleFunc("PUT /api/content/{path...}", s.handleSaveContent)
	mux.HandleFunc("DELETE /api/content/{path...}", s.handleDeleteContent)

	// Build & preview.
	mux.HandleFunc("POST /api/build", s.handleBuild)
	mux.HandleFunc("POST /api/build/validate", s.handleValidate)
	mux.HandleFunc("POST /api/preview/start", s.handlePreviewStart)
	mux.HandleFunc("POST /api/preview/stop", s.handlePreviewStop)

	// Config.
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("PATCH /api/config", s.handleUpdateSettings)
	mux.HandleFunc("GET /api/collections", s.handleGetCollections)

	// Rendering.
	mux.HandleFunc("POST /api/render/markdown", s.handleRenderMarkdown)

	// WebSocket events.
	mux.HandleFunc("/api/events", s.hub.HandleWS)
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

// APIResponse is the standard JSON envelope.
type APIResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

// APIError represents a structured error in the response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}
