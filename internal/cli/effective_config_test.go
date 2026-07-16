package cli

import (
	"testing"
)

func TestRunEffectiveConfig_JSONAndPretty(t *testing.T) {
	dir := createBuildFixtureSite(t)

	for _, format := range []string{"json", "pretty"} {
		cmd := rootCmd
		cmd.SetArgs([]string{"effective-config", dir, "--format", format})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("effective-config --format %s: %v", format, err)
		}
	}
}

func TestRunEffectiveConfig_UnknownFormat(t *testing.T) {
	dir := createBuildFixtureSite(t)

	cmd := rootCmd
	cmd.SetArgs([]string{"effective-config", dir, "--format", "xml"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for unknown format")
	}
}
