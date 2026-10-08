package devlog

import (
	"os"
	"sync/atomic"

	"golang.org/x/term"
)

var colorEnabled atomic.Bool

func init() {
	// Progress follows the terminal alone: FORCE_COLOR asks for colored
	// logs, not for \r redraws in a captured stream.
	progressOK = term.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"

	if os.Getenv("NO_COLOR") != "" {
		return
	}
	if os.Getenv("FORCE_COLOR") != "" {
		colorEnabled.Store(true)
		return
	}
	colorEnabled.Store(term.IsTerminal(int(os.Stderr.Fd())))
}

// SetColor turns ANSI colors on or off, overriding the NO_COLOR, FORCE_COLOR
// and terminal detection done at startup. Tests that compare rendered output
// call SetColor(false) so they pass when run from a color terminal.
func SetColor(on bool) { colorEnabled.Store(on) }

func ansi(code, s string) string {
	if !colorEnabled.Load() {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

func Dim(s string) string     { return ansi("2", s) }
func Bold(s string) string    { return ansi("1", s) }
func Red(s string) string     { return ansi("31", s) }
func Green(s string) string   { return ansi("32", s) }
func Yellow(s string) string  { return ansi("33", s) }
func Blue(s string) string    { return ansi("34", s) }
func Cyan(s string) string    { return ansi("36", s) }
func BgGreen(s string) string { return ansi("42;1", s) }

func statusColor(code int) func(string) string {
	switch {
	case code >= 500:
		return Red
	case code >= 400:
		return Yellow
	case code >= 300:
		return Cyan
	default:
		return Green
	}
}
