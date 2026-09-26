package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/deploy"
)

// deploySite writes a minimal site with a built output directory.
func deploySite(t *testing.T, deployYAML string) string {
	t.Helper()
	dir := t.TempDir()
	yaml := "site:\n  title: Deploy Test\n" + deployYAML
	if err := os.WriteFile(filepath.Join(dir, "sarde.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	dist := filepath.Join(dir, "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<h1>hi</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// runDeployCLI runs `sarde deploy` with every flag spelled out (the cobra
// root is shared across tests, so flag values would otherwise leak) and
// returns stdout split into lines.
func runDeployCLI(t *testing.T, args ...string) ([]string, error) {
	t.Helper()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	defer rootCmd.SetOut(nil)
	rootCmd.SetArgs(append([]string{"deploy"}, args...))
	err := rootCmd.Execute()
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines, err
}

func decodeEvents(t *testing.T, lines []string) []map[string]any {
	t.Helper()
	events := make([]map[string]any, 0, len(lines))
	for _, l := range lines {
		var ev map[string]any
		if err := json.Unmarshal([]byte(l), &ev); err != nil {
			t.Fatalf("stdout line is not JSON: %q (%v)", l, err)
		}
		events = append(events, ev)
	}
	return events
}

func TestDeployJSON_CustomProviderEventStream(t *testing.T) {
	dir := deploySite(t, "deploy:\n  provider: custom\n  command: echo shipped-by-command\n")
	lines, err := runDeployCLI(t, dir, "--format", "json", "--check=false", "--provider", "")
	if err != nil {
		t.Fatalf("deploy: %v\n%s", err, strings.Join(lines, "\n"))
	}
	events := decodeEvents(t, lines)
	for _, ev := range events {
		if ev["v"] != float64(1) {
			t.Errorf("event without v=1: %v", ev)
		}
	}
	if events[0]["event"] != "start" || events[0]["provider"] != "custom" {
		t.Errorf("first event = %v", events[0])
	}
	last := events[len(events)-1]
	if last["event"] != "result" || last["ok"] != true {
		t.Errorf("last event = %v", last)
	}
	sawLog := false
	for _, ev := range events {
		if ev["event"] == "log" && strings.Contains(ev["message"].(string), "shipped-by-command") {
			sawLog = true
		}
	}
	if !sawLog {
		t.Errorf("the command's output did not arrive as a log event: %v", lines)
	}
}

func netlifyHook(t *testing.T, h http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	deployOptionsHook = func(o *deploy.Options) {
		o.BaseURL = srv.URL
		o.Getenv = func(k string) string {
			if k == "NETLIFY_AUTH_TOKEN" {
				return "tok"
			}
			return ""
		}
	}
	t.Cleanup(func() { deployOptionsHook = nil })
}

func TestDeployJSON_AuthFailureEnvelope(t *testing.T) {
	netlifyHook(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Access Denied"}`))
	})
	dir := deploySite(t, "deploy:\n  provider: netlify\n  site_id: site-1\n")
	lines, err := runDeployCLI(t, dir, "--format", "json", "--check=false", "--provider", "")
	if err == nil {
		t.Fatal("expected a non-zero exit")
	}
	events := decodeEvents(t, lines)
	last := events[len(events)-1]
	env, ok := last["error"].(map[string]any)
	if !ok {
		t.Fatalf("last line is not an error envelope: %v", last)
	}
	if env["kind"] != "deploy_failed" || env["code"] != "auth" || !strings.Contains(env["message"].(string), "Access Denied") {
		t.Errorf("envelope = %v", env)
	}
}

func TestDeployJSON_Check(t *testing.T) {
	netlifyHook(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sites/site-1" {
			t.Errorf("--check must only look the site up, got %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{"id":"site-1","name":"docs","ssl_url":"https://docs.example.com"}`))
	})
	// No dist directory needed for a check.
	dir := deploySite(t, "deploy:\n  provider: netlify\n  site_id: site-1\n")
	os.RemoveAll(filepath.Join(dir, "dist"))
	lines, err := runDeployCLI(t, dir, "--format", "json", "--check", "--provider", "")
	if err != nil {
		t.Fatalf("check: %v %v", err, lines)
	}
	events := decodeEvents(t, lines)
	last := events[len(events)-1]
	if last["event"] != "check" || last["target"] != "docs" || last["url"] != "https://docs.example.com" {
		t.Errorf("check event = %v", last)
	}
}

func TestDeployJSON_CheckUnsupportedForCustom(t *testing.T) {
	dir := deploySite(t, "deploy:\n  provider: custom\n  command: echo x\n")
	lines, err := runDeployCLI(t, dir, "--format", "json", "--check", "--provider", "")
	if err == nil {
		t.Fatal("expected an error")
	}
	events := decodeEvents(t, lines)
	env := events[len(events)-1]["error"].(map[string]any)
	if env["kind"] != "deploy_check_failed" {
		t.Errorf("envelope = %v", env)
	}
}
