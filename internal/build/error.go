package build

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// BuildError holds structured information about a build failure.
type BuildError struct {
	Type    string // "template", "frontmatter", "markdown", "build"
	File    string // relative file path
	Line    int    // 1-based line number
	Col     int    // 1-based column number
	Message string // error message
	Frame   string // ±3 lines of source context
}

// Go template errors: template: <name>:<line>:<col>: <message>
var templateErrRe = regexp.MustCompile(`template: [^:]+:(\d+):(\d+): (.+)`)

// Go template errors (without col): template: <name>:<line>: <message>
var templateErrNoColRe = regexp.MustCompile(`template: [^:]+:(\d+): (.+)`)

// YAML errors: yaml: line <n>: <message>
var yamlErrRe = regexp.MustCompile(`yaml: line (\d+): (.+)`)

// Builder wrapped errors: rendering <path> (template <name>): <underlying>
var renderErrRe = regexp.MustCompile(`rendering ([^ ]+) \(template ([^)]+)\): (.+)`)

// ParseBuildError extracts structured error info from a build error.
// Returns nil if the error cannot be parsed into a structured form.
func ParseBuildError(err error, projectDir string) *BuildError {
	if err == nil {
		return nil
	}
	msg := err.Error()

	// Try to extract the source file from wrapped builder errors.
	var sourceFile string
	if m := renderErrRe.FindStringSubmatch(msg); m != nil {
		sourceFile = m[1]
		msg = m[3] // unwrap to underlying error
	}

	// Template error with line:col
	if m := templateErrRe.FindStringSubmatch(msg); m != nil {
		line, _ := strconv.Atoi(m[1])
		col, _ := strconv.Atoi(m[2])
		be := &BuildError{
			Type:    "template",
			Line:    line,
			Col:     col,
			Message: m[3],
		}
		// Try to find the template file
		be.File = resolveErrorFile(sourceFile, projectDir)
		be.Frame = extractFrame(be.File, projectDir, line)
		return be
	}

	// Template error without col
	if m := templateErrNoColRe.FindStringSubmatch(msg); m != nil {
		line, _ := strconv.Atoi(m[1])
		be := &BuildError{
			Type:    "template",
			Line:    line,
			Message: m[2],
		}
		be.File = resolveErrorFile(sourceFile, projectDir)
		be.Frame = extractFrame(be.File, projectDir, line)
		return be
	}

	// YAML error
	if m := yamlErrRe.FindStringSubmatch(msg); m != nil {
		line, _ := strconv.Atoi(m[1])
		be := &BuildError{
			Type:    "frontmatter",
			File:    resolveErrorFile(sourceFile, projectDir),
			Line:    line,
			Message: m[2],
		}
		be.Frame = extractFrame(be.File, projectDir, line)
		return be
	}

	// Fallback: return a generic build error with the message
	return &BuildError{
		Type:    "build",
		File:    resolveErrorFile(sourceFile, projectDir),
		Message: err.Error(),
	}
}

// resolveErrorFile returns a relative file path, preferring the source file if available.
func resolveErrorFile(sourceFile, projectDir string) string {
	if sourceFile == "" {
		return ""
	}
	// If it's already relative, return as-is
	if !strings.HasPrefix(sourceFile, "/") && !strings.Contains(sourceFile, ":\\") {
		return sourceFile
	}
	// Try to make relative to project dir
	if projectDir != "" && strings.HasPrefix(sourceFile, projectDir) {
		rel := strings.TrimPrefix(sourceFile, projectDir)
		rel = strings.TrimPrefix(rel, "/")
		rel = strings.TrimPrefix(rel, "\\")
		return strings.ReplaceAll(rel, "\\", "/")
	}
	return sourceFile
}

// extractFrame reads the source file and returns ±3 lines around the error line.
// The error line is marked with ">" prefix.
func extractFrame(file, projectDir string, line int) string {
	if file == "" || line <= 0 {
		return ""
	}

	// Try to read the file (absolute or relative to projectDir)
	var data []byte
	var err error
	data, err = os.ReadFile(file)
	if err != nil && projectDir != "" {
		data, err = os.ReadFile(projectDir + "/" + file)
	}
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	start := line - 4 // 3 lines before (0-indexed: line-1 minus 3)
	if start < 0 {
		start = 0
	}
	end := line + 3 // 3 lines after
	if end > len(lines) {
		end = len(lines)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		lineNum := i + 1
		marker := "  "
		if lineNum == line {
			marker = "> "
		}
		sb.WriteString(fmt.Sprintf("%s%4d | %s\n", marker, lineNum, lines[i]))
	}
	return sb.String()
}
