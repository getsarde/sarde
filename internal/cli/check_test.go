package cli

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, run func() error) (string, error) {
	t.Helper()
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
	execErr := run()
	w.Close()
	os.Stdout = old
	return string(<-outCh), execErr
}

func TestCheckLinks_SilencesUsage(t *testing.T) {
	if !checkLinksCmd.SilenceUsage {
		t.Fatal("check-links must set SilenceUsage like build/validate")
	}
}

// A failure (here: no project) in --report json mode is an {"error": ...}
// envelope on stdout, the same contract Studio parses for build/validate.
func TestCheckLinks_JSONErrorEnvelope(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(dir+"/sarde.yaml", []byte("plugins:\n  enabled: [no_such_plugin]\n"), 0o644)
	out, err := captureStdout(t, func() error {
		cmd := rootCmd
		cmd.SetArgs([]string{"check-links", dir, "--report", "json"})
		return cmd.Execute()
	})
	if err == nil {
		t.Fatal("expected an error for an invalid config")
	}
	if !strings.Contains(out, `"error"`) || !strings.Contains(out, `"kind"`) {
		t.Fatalf("stdout is not a JSON error envelope: %q", out)
	}
	if strings.Contains(out, "Usage:") {
		t.Fatalf("usage text leaked into stdout: %q", out)
	}
}

func TestCheckLinks_PrettyErrorHasNoEnvelope(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(dir+"/sarde.yaml", []byte("plugins:\n  enabled: [no_such_plugin]\n"), 0o644)
	// Flags persist on the shared rootCmd between tests; clear the previous
	// test's --report json.
	_ = checkLinksCmd.Flags().Set("report", "")
	out, err := captureStdout(t, func() error {
		cmd := rootCmd
		cmd.SetArgs([]string{"check-links", dir})
		return cmd.Execute()
	})
	if err == nil {
		t.Fatal("expected an error for an invalid config")
	}
	if strings.Contains(out, `{"error"`) {
		t.Fatalf("pretty mode must not print an envelope: %q", out)
	}
}
