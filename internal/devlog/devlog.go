package devlog

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func timestamp() string {
	return time.Now().Format("15:04:05")
}

// Log prints a tagged info log line: "15:04:05 [tag] message"
func Log(tag, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s %s\n", Dim(timestamp()), Blue("["+tag+"]"), msg)
}

// Warn prints a warning log line: "15:04:05 [WARN] [tag] message"
func Warn(tag, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s %s %s\n", Bold(timestamp()), Yellow("[WARN]"), Yellow("["+tag+"]"), msg)
}

// Error prints an error log line: "15:04:05 [ERROR] [tag] message"
func Error(tag, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s %s %s\n", Bold(timestamp()), Red("[ERROR]"), Red("["+tag+"]"), msg)
}

// Banner prints the server startup banner without a timestamp prefix.
func Banner(version string, url string, hostFlag string, duration time.Duration) {
	ms := duration.Milliseconds()
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
	fmt.Fprintf(os.Stderr, "%s %s %s%s %s\n",
		Dim(timestamp()),
		color(fmt.Sprintf("[%d]", status)),
		methodStr,
		path,
		Dim(fmt.Sprintf("%dms", ms)),
	)
}
