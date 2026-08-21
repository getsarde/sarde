package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/validate"
)

func decodeEnvelope(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	env, ok := doc["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error envelope: %s", buf.String())
	}
	return env
}

func TestWriteJSONErrorValidation(t *testing.T) {
	verr := &config.ValidationError{Errors: []validate.Error{
		{
			Path:    "plugins.enabled[20]",
			Value:   "text_highlighter",
			Message: "must be one of: text-highlighter, katex",
			Allowed: []string{"text-highlighter", "katex"},
		},
	}}
	// Wrapped the way resolveAll wraps it — errors.As must still find it.
	wrapped := fmt.Errorf("resolving config: %w", verr)

	var buf bytes.Buffer
	writeJSONError(&buf, "build_failed", wrapped)
	env := decodeEnvelope(t, &buf)

	if env["kind"] != "config_validation" {
		t.Errorf("kind = %v, want config_validation", env["kind"])
	}
	details, ok := env["details"].([]any)
	if !ok || len(details) != 1 {
		t.Fatalf("details = %v, want 1 entry", env["details"])
	}
	d := details[0].(map[string]any)
	if d["path"] != "plugins.enabled[20]" || d["value"] != "text_highlighter" {
		t.Errorf("detail = %v", d)
	}
	allowed, ok := d["allowed"].([]any)
	if !ok || len(allowed) != 2 {
		t.Errorf("allowed = %v, want 2 entries", d["allowed"])
	}
}

func TestWriteJSONErrorGeneric(t *testing.T) {
	var buf bytes.Buffer
	writeJSONError(&buf, "dev_failed", fmt.Errorf("port 4727 already in use"))
	env := decodeEnvelope(t, &buf)

	if env["kind"] != "dev_failed" {
		t.Errorf("kind = %v, want dev_failed", env["kind"])
	}
	if !strings.Contains(env["message"].(string), "port 4727") {
		t.Errorf("message = %v", env["message"])
	}
	if _, present := env["details"]; present {
		t.Errorf("details should be omitted for non-validation errors, got %v", env["details"])
	}
}

func TestValidationErrorMessageUnchanged(t *testing.T) {
	// Pretty-mode output contract: the typed error's string form must equal
	// the previous flattened fmt.Errorf form exactly.
	errs := []validate.Error{
		{Path: "a.b", Value: "x", Message: "must be one of: y, z"},
		{Path: "c.d", Message: "is required"},
	}
	typed := &config.ValidationError{Errors: errs}
	legacy := fmt.Sprintf("config validation failed:\n%s", validate.FormatErrors(errs))
	if typed.Error() != legacy {
		t.Errorf("Error() drifted from legacy format:\n got: %q\nwant: %q", typed.Error(), legacy)
	}
}
