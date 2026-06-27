package codediff

import "github.com/yuin/goldmark"

// Extension is a stub. Code diff rendering is handled by the Chroma
// code block renderer's ins={}/del={} line highlighting support.
type Extension struct{}

func (e *Extension) Extend(m goldmark.Markdown) {}
