package cli

import (
	"os"
	"testing"

	"github.com/getsarde/sarde/internal/devlog"
)

// Several tests compare rendered output as plain text. Color detection runs
// once at startup and would add ANSI codes under a color terminal or
// FORCE_COLOR, so turn it off for the whole package.
func TestMain(m *testing.M) {
	devlog.SetColor(false)
	os.Exit(m.Run())
}
