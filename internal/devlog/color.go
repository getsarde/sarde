package devlog

import (
	"os"

	"golang.org/x/term"
)

var colorEnabled bool

func init() {
	// Progress follows the terminal alone: FORCE_COLOR asks for colored
	// logs, not for \r redraws in a captured stream.
	progressOK = term.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"

	if os.Getenv("NO_COLOR") != "" {
		colorEnabled = false
		return
	}
	if os.Getenv("FORCE_COLOR") != "" {
		colorEnabled = true
		return
	}
	colorEnabled = term.IsTerminal(int(os.Stderr.Fd()))
}

func ansi(code, s string) string {
	if !colorEnabled {
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
