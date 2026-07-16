package cli

import (
	"encoding/json"
	"io"
	"os"
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
