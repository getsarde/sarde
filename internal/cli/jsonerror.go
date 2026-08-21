package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/validate"
)

// errorEnvelope is the machine-readable failure shape emitted on stdout when
// --format json is in effect, wrapped as {"error": {...}}. Consumed by Sarde
// Studio's sidecar bridge; keep the field names stable.
type errorEnvelope struct {
	Kind    string           `json:"kind"`
	Message string           `json:"message"`
	Details []validate.Error `json:"details,omitempty"`
}

// emitJSONError writes {"error": {kind, message, details?}} to stdout and
// returns err unchanged so the process still exits non-zero. Config
// validation failures are detected via errors.As and carry structured
// per-field details; any other error is kind/message only.
func emitJSONError(kind string, err error) error {
	writeJSONError(os.Stdout, kind, err)
	return err
}

func writeJSONError(w io.Writer, kind string, err error) {
	env := errorEnvelope{Kind: kind, Message: err.Error()}
	var ve *config.ValidationError
	if errors.As(err, &ve) {
		env.Kind = "config_validation"
		env.Message = "config validation failed"
		env.Details = ve.Errors
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"error": env})
}
