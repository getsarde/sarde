package server

import (
	"net/http"
	"time"
)

// Limits shared by the dev server and the API server. Neither sets a
// ReadTimeout or WriteTimeout: API calls such as build and deploy run long,
// and the dev server holds live-reload WebSocket connections open.
const (
	// readHeaderTimeout bounds how long a client may take to send its
	// request headers, so a stalled connection cannot hold a goroutine
	// forever.
	readHeaderTimeout = 10 * time.Second
	// idleTimeout closes a keep-alive connection that sends no further
	// request.
	idleTimeout = 120 * time.Second
	// maxRequestBody caps an API request body. The largest legitimate
	// bodies are Markdown documents sent for preview.
	maxRequestBody = 10 << 20
)

// bodyLimitMiddleware caps every request body at maxRequestBody. A handler
// that decodes a larger body gets an error from the reader and answers with
// its usual bad-request response.
func bodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		}
		next.ServeHTTP(w, r)
	})
}
