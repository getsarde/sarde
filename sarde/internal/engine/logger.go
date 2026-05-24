package engine

import "sync"

// BuildLogger collects log messages from the build pipeline and plugins.
// Thread-safe for use from parallel BuildDone plugin goroutines.
type BuildLogger struct {
	mu       sync.Mutex
	messages []BuildLogEntry
}

// NewBuildLogger creates an empty BuildLogger.
func NewBuildLogger() *BuildLogger {
	return &BuildLogger{}
}

// Log records a message from the given source.
func (l *BuildLogger) Log(source, message string) {
	l.mu.Lock()
	l.messages = append(l.messages, BuildLogEntry{Source: source, Message: message})
	l.mu.Unlock()
}

// Messages returns all collected log entries.
func (l *BuildLogger) Messages() []BuildLogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]BuildLogEntry, len(l.messages))
	copy(out, l.messages)
	return out
}
