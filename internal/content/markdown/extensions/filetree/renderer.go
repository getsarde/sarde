package filetree

import (
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
	"github.com/getsarde/sarde/internal/content/markdown/icons"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type filetreeRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &filetreeRenderer{} }

func (r *filetreeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindFileTreeBlock, r.render)
}

func (r *filetreeRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	_, _ = w.WriteString("<div class=\"sarde-file-tree\">\n")

	// Extract entries from child AST list nodes
	renderList(w, node, source)

	_, _ = w.WriteString("</div>\n")

	return ast.WalkSkipChildren, nil
}

func renderList(w util.BufWriter, parent ast.Node, source []byte) {
	// Find List children
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Kind() == ast.KindList {
			_, _ = w.WriteString("<ul class=\"sarde-file-tree-list\">\n")
			renderListItems(w, c, source)
			_, _ = w.WriteString("</ul>\n")
		}
	}
	// If no list found, render direct text-based entries
	if parent.FirstChild() == nil || parent.FirstChild().Kind() != ast.KindList {
		// Fallback: extract text and parse manually
		text := extractAllText(parent, source)
		if text != "" {
			lines := strings.Split(text, "\n")
			_, _ = w.WriteString("<ul class=\"sarde-file-tree-list\">\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if strings.HasPrefix(line, "- ") {
					line = line[2:]
				}
				isFolder := strings.HasSuffix(line, "/")
				name := strings.TrimSuffix(line, "/")
				renderItem(w, name, isFolder)
			}
			_, _ = w.WriteString("</ul>\n")
		}
	}
}

func renderListItems(w util.BufWriter, listNode ast.Node, source []byte) {
	for item := listNode.FirstChild(); item != nil; item = item.NextSibling() {
		if item.Kind() != ast.KindListItem {
			continue
		}

		// Get text content of this list item
		name := extractDirectText(item, source)
		name = strings.TrimSpace(name)

		isFolder := strings.HasSuffix(name, "/")
		if isFolder {
			name = strings.TrimSuffix(name, "/")
		}

		// Check if this item has a nested list (subfolder)
		var nestedList ast.Node
		for c := item.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Kind() == ast.KindList {
				nestedList = c
				isFolder = true
				break
			}
		}

		itemClass := "sarde-file-tree-item"
		if isFolder {
			itemClass += " sarde-file-tree-folder"
		} else {
			itemClass += " sarde-file-tree-file"
		}

		icon := getIcon(isFolder, name)
		nameClass := "sarde-file-tree-name"
		if isFolder {
			nameClass += " folder-name"
		}

		_, _ = w.WriteString("<li class=\"" + itemClass + "\">\n")
		_, _ = w.WriteString("<span class=\"sarde-file-tree-entry\">")
		_, _ = w.WriteString(icon)
		_, _ = w.WriteString("<span class=\"" + nameClass + "\">" + htmlutil.EscapeHTML(name) + "</span>")
		_, _ = w.WriteString("</span>\n")

		if nestedList != nil {
			_, _ = w.WriteString("<ul class=\"sarde-file-tree-list\">\n")
			renderListItems(w, nestedList, source)
			_, _ = w.WriteString("</ul>\n")
		}

		_, _ = w.WriteString("</li>\n")
	}
}

func renderItem(w util.BufWriter, name string, isFolder bool) {
	itemClass := "sarde-file-tree-item"
	if isFolder {
		itemClass += " sarde-file-tree-folder"
	} else {
		itemClass += " sarde-file-tree-file"
	}
	icon := getIcon(isFolder, name)
	nameClass := "sarde-file-tree-name"
	if isFolder {
		nameClass += " folder-name"
	}
	_, _ = w.WriteString("<li class=\"" + itemClass + "\">\n")
	_, _ = w.WriteString("<span class=\"sarde-file-tree-entry\">")
	_, _ = w.WriteString(icon)
	_, _ = w.WriteString("<span class=\"" + nameClass + "\">" + htmlutil.EscapeHTML(name) + "</span>")
	_, _ = w.WriteString("</span>\n")
	_, _ = w.WriteString("</li>\n")
}

// extractDirectText gets text from the first paragraph of a node, not from nested lists
func extractDirectText(n ast.Node, source []byte) string {
	var sb strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Kind() == ast.KindList {
			continue // skip nested lists
		}
		extractTextRecursive(c, source, &sb)
	}
	return sb.String()
}

func extractTextRecursive(n ast.Node, source []byte, sb *strings.Builder) {
	if n.Kind() == ast.KindText {
		t := n.(*ast.Text)
		sb.Write(t.Segment.Value(source))
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		extractTextRecursive(c, source, sb)
	}
}

func extractAllText(n ast.Node, source []byte) string {
	var sb strings.Builder
	extractTextRecursive(n, source, &sb)
	return sb.String()
}

func getIcon(isFolder bool, name string) string {
	if isFolder {
		return icons.GetWithClass("folder", "sarde-file-tree-icon folder")
	}
	cls := "sarde-file-tree-icon file"
	if ext := fileExtClass(name); ext != "" {
		cls += " " + ext
	}
	return icons.GetWithClass("file", cls)
}

func fileExtClass(name string) string {
	ext := strings.TrimPrefix(filepath.Ext(name), ".")
	if ext == "" {
		return ""
	}
	for _, c := range ext {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return ""
		}
	}
	return "file-" + ext
}

