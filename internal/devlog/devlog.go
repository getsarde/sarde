package devlog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	mu       sync.Mutex
	out      io.Writer = os.Stderr
	progress string    // current \r-overwrite line, empty when inactive

	// progressOK gates the in-place progress line. It is true only when
	// stderr is an interactive terminal: \r and \033[K garble CI logs, pipes
	// and the desktop app's log panel. Set in init (color.go); guarded by mu.
	progressOK bool

	quiet     atomic.Bool
	warnCount atomic.Int64
)

func timestamp() string { return time.Now().Format("15:04:05") }

// SetQuiet silences informational output (Log and the progress line).
// Warnings and errors still print.
func SetQuiet(q bool) { quiet.Store(q) }

// Quiet reports whether informational output is silenced.
func Quiet() bool { return quiet.Load() }

// DisableProgress turns the in-place progress line off for the rest of the
// process, e.g. when stdout carries machine-readable output.
func DisableProgress() {
	mu.Lock()
	if progress != "" {
		fmt.Fprint(out, "\r\033[K")
		progress = ""
	}
	progressOK = false
	mu.Unlock()
}

// WarnCount returns the number of Warn calls made so far in this process.
// Callers measure a span (such as one build) by taking the difference
// between two readings.
func WarnCount() int64 { return warnCount.Load() }

// DisplayPath shortens an absolute path to one relative to the working
// directory, with forward slashes, so terminals can link it. Relative paths
// and paths outside the working directory are returned unchanged.
func DisplayPath(p string) string {
	if !filepath.IsAbs(p) {
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		return p
	}
	rel, err := filepath.Rel(wd, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return p
	}
	return filepath.ToSlash(rel)
}

// FormatLog returns a formatted log line without printing it.
func FormatLog(tag, msg string) string {
	return fmt.Sprintf("%s %s %s", Dim(timestamp()), Blue("["+tag+"]"), msg)
}

// writeLineUnlocked clears any active progress line, writes a \n-terminated
// line, and redraws the progress line if one was active. Caller must hold mu.
func writeLineUnlocked(line string) {
	if progress != "" {
		fmt.Fprint(out, "\r\033[K")
	}
	fmt.Fprint(out, line)
	if progress != "" {
		fmt.Fprintf(out, "\r%s", progress)
	}
}

// SetProgress sets (or updates) the in-place progress line. The line is
// formatted with FormatLog and written with \r (no newline). Concurrent
// devlog.Log/Warn/Error/Request calls will clear it, write their own line,
// then redraw it, so interleaving is clean. It does nothing when stderr is
// not a terminal or output is quiet.
func SetProgress(tag, format string, args ...any) {
	if quiet.Load() {
		return
	}
	line := FormatLog(tag, fmt.Sprintf(format, args...))
	mu.Lock()
	if progressOK {
		// Clear first: a shorter line would leave the tail of the old one.
		progress = line
		fmt.Fprintf(out, "\r\033[K%s", line)
	}
	mu.Unlock()
}

// ClearProgress removes the in-place progress line from the terminal.
func ClearProgress() {
	mu.Lock()
	if progress != "" {
		progress = ""
		fmt.Fprint(out, "\r\033[K")
	}
	mu.Unlock()
}

// Print writes a preformatted block (one or more lines), keeping any active
// progress line intact below it. A missing trailing newline is added.
func Print(block string) {
	if block == "" {
		return
	}
	if !strings.HasSuffix(block, "\n") {
		block += "\n"
	}
	mu.Lock()
	writeLineUnlocked(block)
	mu.Unlock()
}

// Log prints a tagged info log line: "15:04:05 [tag] message". It does
// nothing when output is quiet.
func Log(tag, format string, args ...any) {
	if quiet.Load() {
		return
	}
	line := FormatLog(tag, fmt.Sprintf(format, args...)) + "\n"
	mu.Lock()
	writeLineUnlocked(line)
	mu.Unlock()
}

// Warn prints a warning log line: "15:04:05 [WARN] [tag] message"
func Warn(tag, format string, args ...any) {
	warnCount.Add(1)
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s %s %s %s\n", Bold(timestamp()), Yellow("[WARN]"), Yellow("["+tag+"]"), msg)
	mu.Lock()
	writeLineUnlocked(line)
	mu.Unlock()
}

// Error prints an error log line: "15:04:05 [ERROR] [tag] message"
func Error(tag, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s %s %s %s\n", Bold(timestamp()), Red("[ERROR]"), Red("["+tag+"]"), msg)
	mu.Lock()
	writeLineUnlocked(line)
	mu.Unlock()
}

// Banner prints the server startup banner without a timestamp prefix.
func Banner(version string, url string, hostFlag string, duration time.Duration) {
	ms := duration.Milliseconds()
	mu.Lock()
	if progress != "" {
		fmt.Fprintf(out, "\r\033[K")
	}
	fmt.Fprintf(out, "\n%s %s %s %d %s\n",
		BgGreen(Bold(" sarde ")),
		Green(version),
		Dim("ready in"),
		ms,
		Dim("ms"),
	)
	fmt.Fprintf(out, "%s Local    %s\n", Dim("┃"), Cyan(url))
	if hostFlag == "" || hostFlag == "127.0.0.1" || hostFlag == "localhost" {
		fmt.Fprintf(out, "%s Network  %s\n\n", Dim("┃"), Dim("use --host 0.0.0.0 to expose"))
	} else {
		fmt.Fprintf(out, "%s Network  %s\n\n", Dim("┃"), Cyan("http://"+hostFlag))
	}
	if progress != "" {
		fmt.Fprintf(out, "\r%s", progress)
	}
	mu.Unlock()
}

// Request prints an HTTP request log line.
// Skips /ws and /favicon.ico.
func Request(method, path string, status int, duration time.Duration) {
	if path == "/ws" || strings.HasSuffix(path, "/favicon.ico") {
		return
	}
	if strings.Contains(path, "/assets/") {
		return
	}
	if status == 304 {
		return
	}
	color := statusColor(status)
	ms := duration.Milliseconds()
	methodStr := ""
	if method != "GET" && method != "" {
		methodStr = color(method) + " "
	}
	line := fmt.Sprintf("%s %s %s%s %s\n",
		Dim(timestamp()),
		color(fmt.Sprintf("[%d]", status)),
		methodStr,
		path,
		Dim(fmt.Sprintf("%dms", ms)),
	)
	mu.Lock()
	writeLineUnlocked(line)
	mu.Unlock()
}
