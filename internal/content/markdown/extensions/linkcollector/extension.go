package linkcollector

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
)

// Extension registers the link collector as a Goldmark AST transformer.
type Extension struct {
	Collector *Collector
}

// Extend implements goldmark.Extender.
func (e *Extension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(e.Collector, 999),
		),
	)
}
