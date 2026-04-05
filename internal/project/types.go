package project

import (
	"time"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

// ProjectState represents the current state of the project manager.
type ProjectState int

const (
	StateClosed    ProjectState = iota
	StateOpen                   // project loaded, ready for operations
	StateBuilding               // build in progress
	StatePreviewing             // dev server running
)

func (s ProjectState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateBuilding:
		return "building"
	case StatePreviewing:
		return "previewing"
	default:
		return "unknown"
	}
}

// ProjectInfo is the summary returned when opening or querying a project.
type ProjectInfo struct {
	Dir         string           `json:"dir"`
	State       string           `json:"state"`
	Title       string           `json:"title"`
	Collections []CollectionInfo `json:"collections"`
}

// CreateOpts configures new project creation.
type CreateOpts struct {
	Title string `json:"title"`
}

// ContentFile is the full representation of a content file for the API.
type ContentFile struct {
	Path        string         `json:"path"`
	Title       string         `json:"title"`
	Collection  string         `json:"collection"`
	Frontmatter map[string]any `json:"frontmatter"`
	Body        string         `json:"body"`
	Draft       bool           `json:"draft"`
	Date        time.Time      `json:"date,omitempty"`
	WordCount   int            `json:"wordCount"`
	ReadingTime int            `json:"readingTime"`
}

// ContentSummary is a lightweight listing entry for content files.
type ContentSummary struct {
	Path   string    `json:"path"`
	Title  string    `json:"title"`
	Draft  bool      `json:"draft"`
	Date   time.Time `json:"date,omitempty"`
	Weight int       `json:"weight"`
}

// CollectionInfo provides metadata about a collection.
type CollectionInfo struct {
	Name      string `json:"name"`
	Title     string `json:"title"`
	Layout    string `json:"layout"`
	PageCount int    `json:"pageCount"`
}

// RenderResult holds the output of a markdown render operation.
type RenderResult struct {
	HTML        string           `json:"html"`
	Headings    []engine.Heading `json:"headings"`
	WordCount   int              `json:"wordCount"`
	ReadingTime int              `json:"readingTime"`
}

// SettingsInput is a partial update to site configuration.
// Only non-nil fields are applied.
type SettingsInput struct {
	Title       *string `json:"title,omitempty"`
	URL         *string `json:"url,omitempty"`
	Language    *string `json:"language,omitempty"`
	Description *string `json:"description,omitempty"`
}
