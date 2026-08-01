package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func TestRunDirectives_JSON(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stdout = w

	// Drain the pipe concurrently: the catalog JSON is larger than the pipe
	// buffer, so encoding would deadlock with no reader.
	outCh := make(chan []byte)
	go func() {
		data, _ := io.ReadAll(r)
		outCh <- data
	}()

	cmd := rootCmd
	cmd.SetArgs([]string{"directives", "--format", "json"})
	execErr := cmd.Execute()

	w.Close()
	os.Stdout = old
	out := <-outCh
	if execErr != nil {
		t.Fatalf("directives --format json: %v", execErr)
	}

	var cat engine.DirectiveCatalog
	if err := json.Unmarshal(out, &cat); err != nil {
		t.Fatalf("output is not valid DirectiveCatalog JSON: %v", err)
	}
	if len(cat.Categories) == 0 {
		t.Fatal("directive catalog JSON has no categories")
	}
}

func TestRunDirectives_Pretty(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{"directives", "--format", "pretty"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("directives --format pretty: %v", err)
	}
}

func TestRunDirectives_UnknownFormat(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{"directives", "--format", "xml"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestRunDirectives_MergesSiteDirectives(t *testing.T) {
	dir := t.TempDir()
	dirDir := filepath.Join(dir, "directives")
	os.MkdirAll(dirDir, 0o755)
	os.WriteFile(filepath.Join(dirDir, "pullquote.yaml"),
		[]byte("name: pullquote\nkind: container\nlabel: Pull Quote\ndescription: A quote\n"), 0o644)
	os.WriteFile(filepath.Join(dirDir, "pullquote.html"),
		[]byte("<blockquote>{{.Body}}</blockquote>"), 0o644)

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stdout = w
	outCh := make(chan []byte)
	go func() {
		data, _ := io.ReadAll(r)
		outCh <- data
	}()

	cmd := rootCmd
	cmd.SetArgs([]string{"directives", "--format", "json", dir})
	execErr := cmd.Execute()

	w.Close()
	os.Stdout = old
	out := <-outCh
	if execErr != nil {
		t.Fatalf("directives --format json <dir>: %v", execErr)
	}

	var cat engine.DirectiveCatalog
	if err := json.Unmarshal(out, &cat); err != nil {
		t.Fatalf("output is not valid DirectiveCatalog JSON: %v", err)
	}

	var found *engine.CatalogDirective
	for _, c := range cat.Categories {
		for i := range c.Directives {
			if c.Directives[i].Name == "pullquote" {
				found = &c.Directives[i]
			}
		}
	}
	if found == nil {
		t.Fatal("site directive pullquote missing from merged catalog")
	}
	if found.Source != "site" {
		t.Errorf("pullquote source = %q, want site", found.Source)
	}
	// Built-in entries must be stamped too.
	if cat.Categories[0].Directives[0].Source != "builtin" {
		t.Errorf("builtin source not stamped: %+v", cat.Categories[0].Directives[0])
	}
}

func TestRunDirectives_MergesPluginDirectives(t *testing.T) {
	dir := t.TempDir()
	plugDir := filepath.Join(dir, "plugins", "callpack")
	os.MkdirAll(filepath.Join(plugDir, "directives"), 0o755)
	os.WriteFile(filepath.Join(plugDir, "plugin.yaml"),
		[]byte("name: CallPack\nslug: callpack\nversion: 1.0.0\n"), 0o644)
	os.WriteFile(filepath.Join(plugDir, "directives", "callout.yaml"),
		[]byte("name: callout\nkind: container\nlabel: Callout\ndescription: A callout\n"), 0o644)
	os.WriteFile(filepath.Join(plugDir, "directives", "callout.html"),
		[]byte("<div>{{.Body}}</div>"), 0o644)

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stdout = w
	outCh := make(chan []byte)
	go func() {
		data, _ := io.ReadAll(r)
		outCh <- data
	}()

	cmd := rootCmd
	cmd.SetArgs([]string{"directives", "--format", "json", dir})
	execErr := cmd.Execute()

	w.Close()
	os.Stdout = old
	out := <-outCh
	if execErr != nil {
		t.Fatalf("directives --format json <dir>: %v", execErr)
	}

	var cat engine.DirectiveCatalog
	if err := json.Unmarshal(out, &cat); err != nil {
		t.Fatalf("output is not valid DirectiveCatalog JSON: %v", err)
	}
	var found *engine.CatalogDirective
	for _, c := range cat.Categories {
		for i := range c.Directives {
			if c.Directives[i].Name == "callout" {
				found = &c.Directives[i]
			}
		}
	}
	if found == nil {
		t.Fatal("plugin directive callout missing from merged catalog")
	}
	if found.Source != "plugin:callpack" {
		t.Errorf("callout source = %q, want plugin:callpack", found.Source)
	}
}

func TestRunDirectives_CheckMalformedPluginDirective(t *testing.T) {
	dir := t.TempDir()
	plugDir := filepath.Join(dir, "plugins", "badpack")
	os.MkdirAll(filepath.Join(plugDir, "directives"), 0o755)
	os.WriteFile(filepath.Join(plugDir, "plugin.yaml"),
		[]byte("name: BadPack\nslug: badpack\nversion: 1.0.0\n"), 0o644)
	os.WriteFile(filepath.Join(plugDir, "directives", "broken.yaml"),
		[]byte("name: broken\nkind: nonsense\nlabel: B\ndescription: d\n"), 0o644)
	os.WriteFile(filepath.Join(plugDir, "directives", "broken.html"),
		[]byte("<div></div>"), 0o644)

	cmd := rootCmd
	cmd.SetArgs([]string{"directives", "--check", dir})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected --check to fail on a malformed plugin directive")
	}
}

func TestRunDirectives_CheckMalformed(t *testing.T) {
	dir := t.TempDir()
	dirDir := filepath.Join(dir, "directives")
	os.MkdirAll(dirDir, 0o755)
	os.WriteFile(filepath.Join(dirDir, "broken.yaml"),
		[]byte("name: broken\nkind: nonsense\nlabel: B\ndescription: d\n"), 0o644)
	os.WriteFile(filepath.Join(dirDir, "broken.html"), []byte("<div></div>"), 0o644)

	cmd := rootCmd
	cmd.SetArgs([]string{"directives", "--check", dir})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected --check to fail on a malformed directive")
	}
}

func TestRunDirectives_CheckClean(t *testing.T) {
	dir := t.TempDir()
	dirDir := filepath.Join(dir, "directives")
	os.MkdirAll(dirDir, 0o755)
	os.WriteFile(filepath.Join(dirDir, "ok.yaml"),
		[]byte("name: ok\nkind: leaf\nlabel: OK\ndescription: d\n"), 0o644)
	os.WriteFile(filepath.Join(dirDir, "ok.html"), []byte("<div>{{.Body}}</div>"), 0o644)

	cmd := rootCmd
	cmd.SetArgs([]string{"directives", "--check", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--check on valid directives failed: %v", err)
	}
}
