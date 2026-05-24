package icons

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed lucide.json
var lucideJSON []byte

// IconifyCollection represents an Iconify JSON icon collection.
type IconifyCollection struct {
	Prefix  string                      `json:"prefix"`
	Width   int                         `json:"width"`
	Height  int                         `json:"height"`
	Icons   map[string]IconifyIcon      `json:"icons"`
	Aliases map[string]IconifyIconAlias `json:"aliases"`
}

// IconifyIcon represents a single icon in an Iconify collection.
type IconifyIcon struct {
	Body   string `json:"body"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// IconifyIconAlias maps an alias name to its parent icon.
type IconifyIconAlias struct {
	Parent string `json:"parent"`
}

// resolver holds loaded icon collections.
var resolver struct {
	once        sync.Once
	collections map[string]*IconifyCollection
}

func init() {
	resolver.collections = make(map[string]*IconifyCollection)
}

func ensureLoaded() {
	resolver.once.Do(func() {
		var col IconifyCollection
		if err := json.Unmarshal(lucideJSON, &col); err == nil {
			resolver.collections["lucide"] = &col
		}
	})
}

// Get returns an inline SVG for the given icon name.
// Supports "set:name" (e.g. "lucide:rocket") or bare "name" (defaults to lucide).
// Returns empty string if the icon is not found.
func Get(name string) string {
	return GetWithClass(name, "")
}

// GetWithClass returns an inline SVG with an optional CSS class attribute.
func GetWithClass(name, class string) string {
	ensureLoaded()

	setName, iconName := parseIconID(name)

	col, ok := resolver.collections[setName]
	if !ok {
		return ""
	}

	icon, ok := col.Icons[iconName]
	if !ok {
		// Try Iconify collection aliases (e.g. "home" -> "house")
		if resolved := resolveIconifyAlias(col, iconName); resolved != "" {
			icon, ok = col.Icons[resolved]
		}
		if !ok {
			// Try custom shorthand aliases
			iconName = resolveAlias(iconName)
			icon, ok = col.Icons[iconName]
			if !ok {
				// Custom alias target might itself be an Iconify alias
				if resolved := resolveIconifyAlias(col, iconName); resolved != "" {
					icon, ok = col.Icons[resolved]
				}
				if !ok {
					return ""
				}
			}
		}
	}

	w := col.Width
	h := col.Height
	if icon.Width > 0 {
		w = icon.Width
	}
	if icon.Height > 0 {
		h = icon.Height
	}
	if w == 0 {
		w = 24
	}
	if h == 0 {
		h = 24
	}

	classAttr := ""
	if class != "" {
		classAttr = fmt.Sprintf(` class="%s"`, class)
	}

	return fmt.Sprintf(
		`<svg%s xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 %d %d">%s</svg>`,
		classAttr, w, h, icon.Body,
	)
}

// parseIconID splits "set:name" into (set, name). Bare "name" defaults to "lucide".
func parseIconID(id string) (string, string) {
	if idx := strings.Index(id, ":"); idx > 0 {
		return id[:idx], id[idx+1:]
	}
	return "lucide", id
}

// aliasMap maps common shorthand names to their Lucide equivalents.
var aliasMap = map[string]string{
	"arrow":      "arrow-right",
	"external":   "external-link",
	"close":      "x",
	"cross":      "x",
	"warning":    "alert-triangle",
	"danger":     "alert-octagon",
	"error":      "x-circle",
	"success":    "check-circle",
	"note":       "info",
	"tip":        "lightbulb",
	"caution":    "alert-triangle",
	"question":   "help-circle",
	"cog":        "settings",
	"gear":       "settings",
	"bolt":       "zap",
	"lightning":  "zap",
	"document":   "file-text",
	"docs":       "book-open",
	"learn":      "graduation-cap",
	"education":  "school",
	"edit":       "pencil",
	"pencil":     "pencil-line",
}

// resolveIconifyAlias follows the alias chain in the Iconify collection
// to find the canonical icon name. Returns empty string if not an alias.
func resolveIconifyAlias(col *IconifyCollection, name string) string {
	if col.Aliases == nil {
		return ""
	}
	// Follow alias chain (max 10 hops to prevent cycles)
	current := name
	for i := 0; i < 10; i++ {
		alias, ok := col.Aliases[current]
		if !ok {
			break
		}
		current = alias.Parent
		// Check if the parent is an actual icon
		if _, ok := col.Icons[current]; ok {
			return current
		}
	}
	return ""
}

func resolveAlias(name string) string {
	if alias, ok := aliasMap[name]; ok {
		return alias
	}
	return name
}

// LoadCollection loads an additional Iconify JSON collection from raw JSON bytes.
func LoadCollection(data []byte) error {
	var col IconifyCollection
	if err := json.Unmarshal(data, &col); err != nil {
		return err
	}
	if col.Prefix == "" {
		return fmt.Errorf("icon collection missing prefix")
	}
	ensureLoaded()
	resolver.collections[col.Prefix] = &col
	return nil
}
