package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// A page on a domain that re-resolves to this machine (DNS rebinding) sends
// its own domain as Host; the dev server must not serve it the site.
func TestDevServer_RefusesAForeignHost(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "index.html"), []byte("<html><body>draft</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	ds := New(Options{ProjectDir: t.TempDir(), OutputDir: out})
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "rebind.attacker.example:1313"
	w := httptest.NewRecorder()
	ds.server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d for a foreign Host, want 403", w.Code)
	}
}

// The guard leaves the usual ways in: loopback, the machine's IP (LAN access
// with --host 0.0.0.0), *.localhost, the bound host, and a client with no Host.
func TestDevServer_ServesLocalAndIPHosts(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "index.html"), []byte("<html><body>ok</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	ds := New(Options{ProjectDir: t.TempDir(), OutputDir: out})
	for _, host := range []string{"127.0.0.1:1313", "localhost:1313", "[::1]:1313", "192.168.1.20:1313"} {
		req := httptest.NewRequest("GET", "/", nil)
		req.Host = host
		w := httptest.NewRecorder()
		ds.server.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Host %q: status %d, want 200", host, w.Code)
		}
	}
}

func TestHostAllowed(t *testing.T) {
	cases := []struct {
		host, bound string
		want        bool
	}{
		{"127.0.0.1:1313", "127.0.0.1", true},
		{"localhost:1313", "127.0.0.1", true},
		{"LOCALHOST", "127.0.0.1", true},
		{"localhost.:1313", "127.0.0.1", true},
		{"site.localhost:1313", "127.0.0.1", true},
		{"[::1]:1313", "127.0.0.1", true},
		{"[::1]", "127.0.0.1", true},
		{"10.0.0.5:1313", "0.0.0.0", true},
		{"", "127.0.0.1", true},
		// A name the server was bound to by --host.
		{"mypc.lan:1313", "mypc.lan", true},
		{"MyPC.lan", "mypc.lan", true},
		// Any other name is a page that rebound its domain to this machine.
		{"rebind.attacker.example:1313", "127.0.0.1", false},
		{"localhost.attacker.example", "127.0.0.1", false},
		{"127.0.0.1.attacker.example", "127.0.0.1", false},
		{"mypc.lan:1313", "0.0.0.0", false},
		{":1313", "127.0.0.1", false},
	}
	for _, c := range cases {
		if got := hostAllowed(c.host, c.bound); got != c.want {
			t.Errorf("hostAllowed(%q, %q) = %v, want %v", c.host, c.bound, got, c.want)
		}
	}
}
