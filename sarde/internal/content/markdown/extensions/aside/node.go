package aside

import (
	gast "github.com/yuin/goldmark/ast"
)

// KindAsideBlock is the NodeKind for AsideBlock.
var KindAsideBlock = gast.NewNodeKind("AsideBlock")

// ValidTypes lists all valid aside types.
var ValidTypes = map[string]bool{
	"note":      true,
	"tip":       true,
	"info":      true,
	"danger":    true,
	"warning":   true,
	"important": true,
	"caution":   true,
	// GitHub-style variants
	"gh-note":      true,
	"gh-tip":       true,
	"gh-important": true,
	"gh-warning":   true,
	"gh-caution":   true,
}

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
