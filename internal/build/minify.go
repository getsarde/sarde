package build

import (
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

// Minifier wraps tdewolff/minify for HTML minification.
type Minifier struct {
	m *minify.M
}

// NewMinifier creates a Minifier configured for HTML.
func NewMinifier() *Minifier {
	m := minify.New()
	m.Add("text/html", &html.Minifier{
		KeepDocumentTags: true,
		KeepEndTags:      true,
	})
	return &Minifier{m: m}
}

// MinifyHTML minifies HTML content. Returns the original on error (non-fatal).
func (mn *Minifier) MinifyHTML(input []byte) []byte {
	out, err := mn.m.Bytes("text/html", input)
	if err != nil {
		return input
	}
	return out
}
