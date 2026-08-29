package cli

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/getsarde/sarde/internal/plugin/catalog"
)

func TestRunPlugins_JSON(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	outCh := make(chan []byte)
	go func() {
		data, _ := io.ReadAll(r)
		outCh <- data
	}()

	cmd := rootCmd
	cmd.SetArgs([]string{"plugins", t.TempDir(), "--format", "json"})
	execErr := cmd.Execute()

	w.Close()
	os.Stdout = old
	out := <-outCh
	if execErr != nil {
		t.Fatalf("plugins --format json: %v", execErr)
	}
	var cat catalog.Catalog
	if err := json.Unmarshal(out, &cat); err != nil {
		t.Fatalf("output is not valid catalog JSON: %v", err)
	}
	if len(cat.Plugins) < 25 {
		t.Fatalf("expected >= 25 plugins, got %d", len(cat.Plugins))
	}
}

func TestRunPlugins_Pretty(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{"plugins", t.TempDir()})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("plugins: %v", err)
	}
}

func TestRunPlugins_UnknownFormat(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{"plugins", t.TempDir(), "--format", "xml"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for unknown format")
	}
}
