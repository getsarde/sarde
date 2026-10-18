package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validateSite writes a site whose one page is valid apart from what
// extraFrontmatter adds.
func validateSite(t *testing.T, extraFrontmatter string) string {
	t.Helper()
	dir := t.TempDir()
	yaml := "site:\n  title: Validate Test\n  url: \"http://localhost:3000\"\n  description: A test site\nbuild:\n  last_updated: mtime\n"
	if err := os.WriteFile(filepath.Join(dir, "sarde.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "content", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	page := "---\ntitle: Getting started\ndescription: First steps\n" + extraFrontmatter + "---\n# Getting started\n\nInstall the tool.\n"
	if err := os.WriteFile(filepath.Join(dir, "content", "docs", "start.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// runValidateCLI runs `sarde validate` with every flag spelled out (the
// cobra root is shared, so values would otherwise leak between tests).
func runValidateCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	defer rootCmd.SetOut(nil)
	rootCmd.SetArgs(append([]string{"validate", "--strict=false", "--lint=false", "--quiet=false"}, args...))
	err := rootCmd.Execute()
	return out.String(), err
}

func decodeValidate(t *testing.T, out string) validateJSON {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("stdout has %d lines, want one JSON object:\n%s", len(lines), out)
	}
	var res validateJSON
	if err := json.Unmarshal([]byte(lines[0]), &res); err != nil {
		t.Fatalf("stdout is not the result object: %v\n%s", err, out)
	}
	return res
}

func TestValidateJSON_Success(t *testing.T) {
	dir := validateSite(t, "")
	out, err := runValidateCLI(t, dir, "--format", "json")
	if err != nil {
		t.Fatalf("validate: %v\n%s", err, out)
	}
	res := decodeValidate(t, out)
	if !res.OK || res.Pages == 0 || res.Warnings == nil || len(res.Warnings) != 0 {
		t.Errorf("result = %+v, want ok with pages and an empty warnings list", res)
	}
	if !strings.Contains(out, `"warnings":[]`) {
		t.Errorf("an empty warnings list must encode as [], got %s", out)
	}
}

func TestValidateJSON_Warnings(t *testing.T) {
	dir := validateSite(t, "layout: not-a-layout\n")
	out, err := runValidateCLI(t, dir, "--format", "json")
	if err != nil {
		t.Fatalf("validate: %v\n%s", err, out)
	}
	res := decodeValidate(t, out)
	if !res.OK || len(res.Warnings) == 0 {
		t.Fatalf("result = %+v, want ok with a warning", res)
	}
	w := res.Warnings[0]
	if !strings.Contains(w.File, "start.md") || w.Level == "" || w.Message == "" {
		t.Errorf("warning = %+v, want the page, a level and a message", w)
	}
}

func TestValidateJSON_StrictFailsWithOneObject(t *testing.T) {
	dir := validateSite(t, "layout: not-a-layout\n")
	out, err := runValidateCLI(t, dir, "--format", "json", "--strict")
	if !errors.Is(err, errStrictWarnings) {
		t.Fatalf("err = %v, want the --strict failure", err)
	}
	// One object with ok false, and no error envelope after it.
	if res := decodeValidate(t, out); res.OK || len(res.Warnings) == 0 {
		t.Errorf("result = %+v, want ok false with the warnings", res)
	}
}

func TestValidateJSON_ConfigErrorKeepsTheEnvelope(t *testing.T) {
	dir := validateSite(t, "")
	cfg := filepath.Join(dir, "sarde.yaml")
	data, _ := os.ReadFile(cfg)
	os.WriteFile(cfg, append(data, []byte("deploy:\n  redirect_format: everywhere\n")...), 0o644)

	// The envelope goes to the process stdout (emitJSONError), not cmd's.
	got, err := captureStdout(t, func() error {
		_, err := runValidateCLI(t, dir, "--format", "json")
		return err
	})
	if err == nil {
		t.Fatal("expected a config error")
	}
	if !strings.Contains(got, `"kind":"config_validation"`) {
		t.Errorf("stdout = %s, want the config_validation envelope", got)
	}
}

func TestValidatePretty_Summary(t *testing.T) {
	dir := validateSite(t, "")
	out, err := runValidateCLI(t, dir, "--format", "pretty")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !strings.Contains(out, "validated in") {
		t.Errorf("pretty output = %q, want the summary line", out)
	}
}
