package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/project"
)

func setupAPITest(t *testing.T) (*APIServer, string) {
	t.Helper()

	// Create a test project.
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "content", "blog"), 0o755)
	os.WriteFile(filepath.Join(dir, "site.yaml"), []byte("site:\n  title: \"API Test\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "content", "_index.md"), []byte("---\ntitle: Home\n---\n# Home\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "content", "blog", "_index.md"), []byte("---\ntitle: Blog\n---\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "content", "blog", "post.md"), []byte("---\ntitle: Post\ndate: \"2025-01-01T00:00:00Z\"\n---\n# Post\n"), 0o644)

	hub := project.NewEventHub()
	pm := project.NewProjectManager(hub, embedded.ThemeFS(), nil)

	srv := NewAPIServer(pm, hub)
	return srv, dir
}

func apiRequest(t *testing.T, srv *APIServer, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	srv.setupRoutes(mux)

	var reqBody *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) APIResponse {
	t.Helper()
	var resp APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parsing response: %v (body: %s)", err, w.Body.String())
	}
	return resp
}

func TestAPI_ProjectOpenAndInfo(t *testing.T) {
	srv, dir := setupAPITest(t)

	// Open project.
	w := apiRequest(t, srv, "POST", "/api/project/open", map[string]any{"dir": dir})
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	resp := parseResponse(t, w)
	if !resp.Success {
		t.Errorf("expected success, got error: %v", resp.Error)
	}

	// Get project info.
	w = apiRequest(t, srv, "GET", "/api/project/info", nil)
	resp = parseResponse(t, w)
	if !resp.Success {
		t.Errorf("expected success, got error: %v", resp.Error)
	}

	// Close.
	w = apiRequest(t, srv, "POST", "/api/project/close", nil)
	resp = parseResponse(t, w)
	if !resp.Success {
		t.Errorf("close failed: %v", resp.Error)
	}
}

func TestAPI_ContentList(t *testing.T) {
	srv, dir := setupAPITest(t)

	// Must open project first.
	apiRequest(t, srv, "POST", "/api/project/open", map[string]any{"dir": dir})

	w := apiRequest(t, srv, "GET", "/api/content?collection=blog", nil)
	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatalf("ListContent failed: %v", resp.Error)
	}
}

func TestAPI_ContentCRUD(t *testing.T) {
	srv, dir := setupAPITest(t)
	apiRequest(t, srv, "POST", "/api/project/open", map[string]any{"dir": dir})

	// Create.
	w := apiRequest(t, srv, "POST", "/api/content", map[string]any{
		"collection": "blog",
		"title":      "New Article",
	})
	if w.Code != 201 {
		t.Fatalf("create status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}

	// Read.
	w = apiRequest(t, srv, "GET", "/api/content/blog/new-article.md", nil)
	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatalf("read failed: %v", resp.Error)
	}

	// Save.
	w = apiRequest(t, srv, "PUT", "/api/content/blog/new-article.md", map[string]any{
		"frontmatter": map[string]any{"title": "Updated Article"},
		"body":        "# Updated\n",
	})
	resp = parseResponse(t, w)
	if !resp.Success {
		t.Fatalf("save failed: %v", resp.Error)
	}

	// Delete.
	w = apiRequest(t, srv, "DELETE", "/api/content/blog/new-article.md", nil)
	resp = parseResponse(t, w)
	if !resp.Success {
		t.Fatalf("delete failed: %v", resp.Error)
	}
}

func TestAPI_Build(t *testing.T) {
	srv, dir := setupAPITest(t)
	apiRequest(t, srv, "POST", "/api/project/open", map[string]any{"dir": dir})

	w := apiRequest(t, srv, "POST", "/api/build", nil)
	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatalf("build failed: %v", resp.Error)
	}
}

func TestAPI_RenderMarkdown(t *testing.T) {
	srv, _ := setupAPITest(t)

	w := apiRequest(t, srv, "POST", "/api/render/markdown", map[string]any{
		"markdown": "# Hello\n\nSome text.",
	})
	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatalf("render failed: %v", resp.Error)
	}
}

func TestAPI_ProjectNotOpen(t *testing.T) {
	srv, _ := setupAPITest(t)

	// Without opening a project, content operations should fail.
	w := apiRequest(t, srv, "GET", "/api/content?collection=blog", nil)
	resp := parseResponse(t, w)
	if resp.Success {
		t.Error("expected error when project not open")
	}
}

func TestAPI_ResponseEnvelope(t *testing.T) {
	srv, _ := setupAPITest(t)

	w := apiRequest(t, srv, "GET", "/api/config", nil)
	resp := parseResponse(t, w)

	// Should be a proper error envelope.
	if resp.Success {
		t.Error("expected error when project not open")
	}
	if resp.Error == nil {
		t.Error("expected error object in response")
	}
	if resp.Error.Code == "" {
		t.Error("expected error code")
	}
}
