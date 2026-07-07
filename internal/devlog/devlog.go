package devlog

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	mu       sync.Mutex
	progress string // current \r-overwrite line, empty when inactive
)

func timestamp() string { return time.Now().Format("15:04:05") }

// FormatLog returns a formatted log line without printing it.
func FormatLog(tag, msg string) string {
	return fmt.Sprintf("%s %s %s", Dim(timestamp()), Blue("["+tag+"]"), msg)
}

// writeLineUnlocked clears any active progress line, writes a \n-terminated
// line, and redraws the progress line if one was active. Caller must hold mu.
func writeLineUnlocked(line string) {
	if progress != "" {
		fmt.Fprintf(os.Stderr, "\r\033[K")
	}
	fmt.Fprint(os.Stderr, line)
	if progress != "" {
		fmt.Fprintf(os.Stderr, "\r%s", progress)
	}
}

// SetProgress sets (or updates) the in-place progress line. The line is
// formatted with FormatLog and written with \r (no newline). Concurrent
// devlog.Log/Warn/Error/Request calls will clear it, write their own line,
// then redraw it, so interleaving is clean.
func SetProgress(tag, format string, args ...any) {
	line := FormatLog(tag, fmt.Sprintf(format, args...))
	mu.Lock()
	progress = line
	fmt.Fprintf(os.Stderr, "\r%s", line)
	mu.Unlock()
}

// ClearProgress removes the in-place progress line from the terminal.
func ClearProgress() {
	mu.Lock()
	progress = ""
	fmt.Fprintf(os.Stderr, "\r\033[K")
	mu.Unlock()
}

// Log prints a tagged info log line: "15:04:05 [tag] message"
func Log(tag, format string, args ...any) {
	line := FormatLog(tag, fmt.Sprintf(format, args...)) + "\n"
	mu.Lock()
	writeLineUnlocked(line)
	mu.Unlock()
}

// Warn prints a warning log line: "15:04:05 [WARN] [tag] message"
func Warn(tag, format string, args ...any) {
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
		fmt.Fprintf(os.Stderr, "\r\033[K")
	}
	fmt.Fprintf(os.Stderr, "\n%s %s %s %d %s\n",
		BgGreen(Bold(" sarde ")),
		Green(version),
		Dim("ready in"),
		ms,
		Dim("ms"),
	)
	fmt.Fprintf(os.Stderr, "%s Local    %s\n", Dim("┃"), Cyan(url))
	if hostFlag == "" || hostFlag == "127.0.0.1" || hostFlag == "localhost" {
		fmt.Fprintf(os.Stderr, "%s Network  %s\n\n", Dim("┃"), Dim("use --host 0.0.0.0 to expose"))
	} else {
		fmt.Fprintf(os.Stderr, "%s Network  %s\n\n", Dim("┃"), Cyan("http://"+hostFlag))
	}
	if progress != "" {
		fmt.Fprintf(os.Stderr, "\r%s", progress)
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
