package syntax

// Diagnostic represents a syntax issue found in a markdown file.
type Diagnostic struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
	Level   string `json:"level"` // "warning" or "error"
}
