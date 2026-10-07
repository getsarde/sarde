package aside

import (
	"github.com/getsarde/sarde/internal/content/markdown/fence"
	gast "github.com/yuin/goldmark/ast"
)

// KindAsideBlock is the NodeKind for AsideBlock.
var KindAsideBlock = gast.NewNodeKind("AsideBlock")

// ValidTypes lists all valid aside types. The list lives in the fence package
// so ":::/aside" and sarde check-syntax recognize the same names.
var ValidTypes = func() map[string]bool {
	m := make(map[string]bool, len(fence.AsideTypes))
	for _, t := range fence.AsideTypes {
		m[t] = true
	}
	return m
}()

// DefaultTitles maps aside types to their default display titles.
var DefaultTitles = map[string]string{
	"note":      "Note",
	"tip":       "Tip",
	"info":      "Info",
	"danger":    "Danger",
	"warning":   "Warning",
	"important": "Important",
	"caution":   "Caution",
	// GitHub-style variants
	"gh-note":      "Note",
	"gh-tip":       "Tip",
	"gh-important": "Important",
	"gh-warning":   "Warning",
	"gh-caution":   "Caution",
}

// AsideBlock is an AST node representing an aside block.
type AsideBlock struct {
	gast.BaseBlock
	AsideType string // "note", "tip", etc.
	Title     string // Custom title or empty for default
	Icon      string // Explicit Lucide icon name (optional; overrides the type icon)
}

// Kind implements ast.Node.Kind.
func (n *AsideBlock) Kind() gast.NodeKind {
	return KindAsideBlock
}

// Dump implements ast.Node.Dump.
func (n *AsideBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Type":  n.AsideType,
		"Title": n.Title,
	}, nil)
}

// GetDisplayTitle returns the custom title or the default title for the type.
func (n *AsideBlock) GetDisplayTitle() string {
	if n.Title != "" {
		return n.Title
	}
	if title, ok := DefaultTitles[n.AsideType]; ok {
		return title
	}
	return "Note"
}
