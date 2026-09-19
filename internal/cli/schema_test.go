package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunSchema_JSONAndPretty(t *testing.T) {
	dir := createBuildFixtureSite(t)
	os.MkdirAll(filepath.Join(dir, "content", "posts"), 0o755)

	for _, format := range []string{"json", "pretty"} {
		cmd := rootCmd
		cmd.SetArgs([]string{"schema", "posts", dir, "--format", format})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("schema --format %s: %v", format, err)
		}
	}
}

func TestRunSchema_UnknownFormat(t *testing.T) {
	dir := createBuildFixtureSite(t)

	cmd := rootCmd
	cmd.SetArgs([]string{"schema", "posts", dir, "--format", "xml"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestRunSchema_UnknownCollection(t *testing.T) {
	dir := createBuildFixtureSite(t)

	cmd := rootCmd
	cmd.SetArgs([]string{"schema", "no-such-collection", dir, "--format", "json"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for unknown collection")
	}
}
