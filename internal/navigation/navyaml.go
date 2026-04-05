package navigation

import (
	"fmt"
	"os"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/engine"
	"gopkg.in/yaml.v3"
)

// navYAMLItem represents a single item in a nav.yaml file.
type navYAMLItem struct {
	Label      string            `yaml:"label"`
	Page       string            `yaml:"page"`       // slug relative to collection
	URL        string            `yaml:"url"`         // external URL
	External   bool              `yaml:"external"`
	Badge      string            `yaml:"badge"`
	BadgeColor string            `yaml:"badge_color"`
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
		node.Weight = page.Weight
		if node.Label == "" {
			if page.SidebarLabel != "" {
				node.Label = page.SidebarLabel
			} else {
				node.Label = page.Title
			}
		}
		// Merge attrs: nav.yaml attrs first, then page sidebar_attrs overrides.
		node.Attrs = mergeAttrs(item.Attrs, copyAttrs(page.Params))
		// Apply badge from nav.yaml (page badge can also be set via frontmatter).
		if item.Badge != "" {
			page.Badge = item.Badge
		}
		if item.BadgeColor != "" {
			page.BadgeColor = item.BadgeColor
		}
	} else if item.URL != "" {
		// External link.
		node.URL = item.URL
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
	return lookup
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
