package navigation

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
	"gopkg.in/yaml.v3"
)

// navYAMLItem represents a single item in a nav.yaml file.
type navYAMLItem struct {
	Label      string            `yaml:"label"`
	Page       string            `yaml:"page"` // slug relative to collection
	URL        string            `yaml:"url"`  // another site, or a site path
	External   bool              `yaml:"external"`
	Badge      engine.Badge      `yaml:"badge"`
	BadgeColor string            `yaml:"badge_color"` // legacy; normalized after decode
	Collapsed  *bool             `yaml:"collapsed"`
	Attrs      map[string]string `yaml:"attrs"`
	Items      []navYAMLItem     `yaml:"items"`
}

// BuildNavTreeFromYAML parses a nav.yaml file and constructs a NavTree.
// When nav.yaml exists, it completely replaces auto-generated navigation.
func BuildNavTreeFromYAML(yamlPath string, collection *engine.Collection) (*engine.NavTree, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("reading nav.yaml: %w", err)
	}

	var items []navYAMLItem
	if err := yaml.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parsing nav.yaml: %w", err)
	}

	// Build page lookup: slug → *Page.
	lookup := buildPageLookup(collection)

	root := &engine.NavNode{
		Label: collection.Title,
		Depth: 0,
	}

	for _, item := range items {
		node := buildNodeFromYAMLItem(item, lookup, 1)
		if node != nil {
			node.Parent = root
			root.Children = append(root.Children, node)
		}
	}

	flat := flattenTree(root)
	for i, node := range flat {
		node.Position = i
	}

	return &engine.NavTree{
		Root:       root,
		Flat:       flat,
		TotalPages: len(flat),
		MaxDepth:   computeMaxDepth(root),
	}, nil
}

// buildNodeFromYAMLItem recursively creates NavNodes from YAML items.
func buildNodeFromYAMLItem(item navYAMLItem, lookup map[string]*engine.Page, depth int) *engine.NavNode {
	node := &engine.NavNode{
		Label: item.Label,
		Depth: depth,
	}

	// Page reference — look up by slug.
	if item.Page != "" {
		page := lookup[item.Page]
		if page == nil {
			// Page not found — skip this item.
			return nil
		}
		node.Page = page
		node.URL = page.RelPermalink
		node.Slug = page.Slug
		node.Order = page.Sidebar.Order
		if node.Label == "" {
			if page.Sidebar.Label != "" {
				node.Label = page.Sidebar.Label
			} else {
				node.Label = page.Title
			}
		}
		// Merge attrs: nav.yaml attrs first, then page sidebar attrs overrides.
		node.Attrs = mergeAttrs(item.Attrs, cloneStringMap(page.Sidebar.Attrs))
		// Badge: page frontmatter badge is the baseline; an explicit nav.yaml
		// badge wins. Stored on the node (not the shared Page) so one tree's
		// nav.yaml badge cannot leak into other trees referencing the page.
		node.Badge = page.Sidebar.Badge
		badge := item.Badge
		if badge.Variant == engine.BadgeVariantDefault && item.BadgeColor != "" {
			if alias, ok := engine.VariantAliases[item.BadgeColor]; ok {
				badge.Variant = alias
			}
		}
		if !badge.IsEmpty() {
			node.Badge = badge
		}
	} else if item.URL != "" {
		// Link by URL: another site, or a site path. Attrs keeps target and rel
		// for themes that render it; External drives the default theme.
		node.URL = item.URL
		node.External = item.External
		if item.External {
			if node.Attrs == nil {
				node.Attrs = make(map[string]string)
			}
			if _, ok := node.Attrs["target"]; !ok {
				node.Attrs["target"] = "_blank"
			}
			if _, ok := node.Attrs["rel"]; !ok {
				node.Attrs["rel"] = "noopener noreferrer"
			}
		}
		if item.Attrs != nil {
			node.Attrs = mergeAttrs(node.Attrs, item.Attrs)
		}
	}
	// Otherwise: group node (label only, no URL).

	// Per-group open control: collapsed: false means open by default.
	if item.Collapsed != nil && !*item.Collapsed {
		node.DefaultOpen = true
	}

	// Process children.
	for _, child := range item.Items {
		childNode := buildNodeFromYAMLItem(child, lookup, depth+1)
		if childNode != nil {
			childNode.Parent = node
			node.Children = append(node.Children, childNode)
		}
	}

	return node
}

// buildPageLookup creates a map from various slug forms to pages.
func buildPageLookup(col *engine.Collection) map[string]*engine.Page {
	lookup := make(map[string]*engine.Page)
	for _, page := range col.Pages {
		if page.Kind == engine.KindSection {
			continue
		}
		// Index by slug.
		lookup[page.Slug] = page
		// Index by relative path within collection (e.g., "guides/auth").
		relPath := strings.TrimPrefix(page.RelPermalink, "/"+col.Name+"/")
		relPath = strings.TrimSuffix(relPath, "/")
		if relPath != "" {
			lookup[relPath] = page
		}
	}
	// Also index by the source file name, which keeps a numeric ordering
	// prefix the URL drops ("guide/01-introduction" for /docs/guide/introduction/).
	// The URL-derived keys win when both forms collide.
	for _, page := range col.Pages {
		if page.Kind == engine.KindSection || page.FilePath == "" {
			continue
		}
		base := filepath.Base(page.FilePath)
		if base == "index.md" || base == "_index.md" {
			continue
		}
		relPath := strings.TrimSuffix(strings.TrimPrefix(page.RelPermalink, "/"+col.Name+"/"), "/")
		if relPath == "" {
			continue
		}
		src := path.Join(path.Dir(relPath), strings.TrimSuffix(base, filepath.Ext(base)))
		if _, taken := lookup[src]; !taken {
			lookup[src] = page
		}
	}
	return lookup
}

// UnresolvedNavRef is a nav.yaml page reference that matches no page.
type UnresolvedNavRef struct {
	Page string
	Line int // 1-based line of the page value
	Col  int // 1-based column of the page value
}

// CheckNavYAML parses a nav.yaml file and returns its page references that
// match no page in col, with their positions. Pass col with every lane's
// pages: a reference that resolves in any language or version is not
// reported. An unresolved entry drops its nested items from the sidebar
// too, so they are not checked. The error is the read or parse error that
// BuildNavTreeFromYAML would return.
func CheckNavYAML(yamlPath string, col *engine.Collection) ([]UnresolvedNavRef, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("reading nav.yaml: %w", err)
	}
	var items []navYAMLItem
	if err := yaml.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parsing nav.yaml: %w", err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing nav.yaml: %w", err)
	}

	lookup := buildPageLookup(col)
	var unresolved []UnresolvedNavRef
	var walk func(n *yaml.Node)
	walk = func(n *yaml.Node) {
		switch n.Kind {
		case yaml.DocumentNode, yaml.SequenceNode:
			for _, c := range n.Content {
				walk(c)
			}
		case yaml.MappingNode:
			var items *yaml.Node
			for i := 0; i+1 < len(n.Content); i += 2 {
				key, val := n.Content[i], n.Content[i+1]
				switch key.Value {
				case "page":
					if val.Value != "" && lookup[val.Value] == nil {
						unresolved = append(unresolved, UnresolvedNavRef{Page: val.Value, Line: val.Line, Col: val.Column})
						return
					}
				case "items":
					items = val
				}
			}
			if items != nil {
				walk(items)
			}
		}
	}
	walk(&doc)
	return unresolved, nil
}

// mergeAttrs merges two attribute maps. Values in `override` take precedence.
func mergeAttrs(base, override map[string]string) map[string]string {
	if base == nil && override == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}
