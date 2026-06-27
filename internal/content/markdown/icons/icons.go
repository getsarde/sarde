package icons

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	swarmicons "github.com/frostybee/go-swarm-icons"
	"github.com/frostybee/go-swarm-icons/lucide"
)

var manager *swarmicons.IconManager

var managerMu sync.Mutex

var defaultPrefix = "lucide"

var setAliases = map[string]string{
	"brands": "simple-icons",
}

var aliasMap = map[string]string{
	"arrow":     "arrow-right",
	"external":  "external-link",
	"close":     "x",
	"cross":     "x",
	"warning":   "alert-triangle",
	"danger":    "alert-octagon",
	"error":     "x-circle",
	"success":   "check-circle",
	"note":      "info",
	"tip":       "lightbulb",
	"caution":   "alert-triangle",
	"question":  "help-circle",
	"cog":       "settings",
	"gear":      "settings",
	"bolt":      "zap",
	"lightning": "zap",
	"document":  "file-text",
	"docs":      "book-open",
	"learn":     "graduation-cap",
	"education": "school",
	"edit":      "pencil",
	"pencil":    "pencil-line",
}

var usedSets sync.Map

var spriteMode atomic.Bool

var sprites *swarmicons.SpriteCollector

var (
	collectionsMu sync.RWMutex
	collections   = make(map[string]*iconifyCollection)
)

type iconifyCollection struct {
	Prefix string    `json:"prefix"`
	Info   *iconInfo `json:"info,omitempty"`
}

type iconInfo struct {
	Name    string          `json:"name"`
	License *iconifyLicense `json:"license,omitempty"`
}

type iconifyLicense struct {
	Title string `json:"title"`
	SPDX  string `json:"spdx"`
	URL   string `json:"url"`
}

func init() {
	ensureManager()
}

func ensureManager() {
	managerMu.Lock()
	defer managerMu.Unlock()
	if manager != nil {
		return
	}

	cfg := swarmicons.NewConfig().
		AddProvider("lucide", lucide.Provider()).
		DefaultPrefix("lucide").
		DefaultAttributes(map[string]string{
			"xmlns": "http://www.w3.org/2000/svg",
		}).
		IgnoreNotFound()

	var err error
	manager, err = cfg.Build()
	if err != nil {
		panic("icons: " + err.Error())
	}

	sprites = swarmicons.NewSpriteCollector()

	collectionsMu.Lock()
	collections["lucide"] = &iconifyCollection{
		Prefix: "lucide",
		Info: &iconInfo{
			Name:    "Lucide",
			License: &iconifyLicense{Title: "ISC License", SPDX: "ISC", URL: "https://github.com/lucide-icons/lucide/blob/main/LICENSE"},
		},
	}
	collectionsMu.Unlock()
}

func extractLicenseInfo(prefix string, data []byte) {
	var col iconifyCollection
	if err := json.Unmarshal(data, &col); err == nil {
		col.Prefix = prefix
		collectionsMu.Lock()
		collections[prefix] = &col
		collectionsMu.Unlock()
	}
}

// SetDefaultPrefix sets the icon set used for bare (prefixless) names.
func SetDefaultPrefix(p string) {
	if p == "" {
		return
	}
	defaultPrefix = p
	manager.SetDefaultPrefix(p)
}

// SetRenderMode selects the icon output mode.
func SetRenderMode(m string) {
	spriteMode.Store(strings.EqualFold(strings.TrimSpace(m), "sprite"))
}

// SpriteMode reports whether sprite output is enabled.
func SpriteMode() bool {
	return spriteMode.Load()
}

// SpriteForHTML scans finished page HTML for sprite <use> references and returns
// a hidden <svg> holding one <symbol> per unique referenced icon.
func SpriteForHTML(pageHTML []byte) []byte {
	if !spriteMode.Load() {
		return nil
	}
	return sprites.SpriteSheet(pageHTML)
}

// Get returns an inline SVG for the given icon name.
func Get(name string) string {
	return GetWithClass(name, "")
}

// GetWithClass returns an inline SVG with an optional CSS class attribute.
// Used by block extensions (aside/card/linkcard/linkbutton). No fallback;
// defaults to 16x16. ARIA attributes (aria-hidden, focusable) are injected
// by the manager's renderer.
func GetWithClass(name, class string) string {
	icon, _, ok := lookupIcon(name)
	if !ok {
		return ""
	}
	return icon.Class(class).Width("16").ToHTML()
}

// Render returns an inline SVG for the template func and inline Markdown extension.
// Falls back to "circle-help" on miss, applies transforms, accessibility attributes.
func Render(name, baseClass string, attrs map[string]string) string {
	ariaAttrs := buildARIAAttrs(attrs)

	icon, spriteID, ok := lookupIcon(name, ariaAttrs)
	if !ok {
		icon, spriteID, ok = lookupIcon("circle-help", ariaAttrs)
		if !ok {
			return ""
		}
	}

	class := baseClass
	var callerW, callerH, userStyle string
	passThrough := map[string]string{}

	for k, v := range attrs {
		switch k {
		case "class":
			if v != "" {
				class = strings.TrimSpace(class + " " + v)
			}
		case "width":
			callerW = v
		case "height":
			callerH = v
		case "style":
			userStyle = v
		case "rotate", "flip", "title":
			// handled separately
		case "role", "aria-label", "aria-labelledby":
			// already applied via manager.Get caller attrs
		default:
			passThrough[k] = v
		}
	}

	if len(passThrough) > 0 {
		icon = icon.Attr(passThrough)
	}
	if class != "" {
		icon = icon.Class(class)
	}

	rawBody := icon.Content()
	viewBox := iconViewBox(icon)

	titleText := attrs["title"]
	if titleText != "" {
		icon = icon.Title(titleText)
	}

	if r := attrs["rotate"]; r != "" {
		if deg, err := strconv.ParseFloat(r, 64); err == nil {
			icon = icon.Rotate(deg)
		}
	}
	if f := attrs["flip"]; f != "" {
		switch strings.ToLower(strings.TrimSpace(f)) {
		case "horizontal,vertical", "vertical,horizontal", "both":
			icon = icon.Flip("h").Flip("v")
		default:
			icon = icon.Flip(f)
		}
	}

	if userStyle != "" {
		existing := icon.Attributes()["style"]
		if existing != "" {
			icon = icon.Attr(map[string]string{"style": existing + "; " + userStyle})
		} else {
			icon = icon.Attr(map[string]string{"style": userStyle})
		}
	}

	switch {
	case callerW != "" && callerH != "":
		icon = icon.Width(callerW)
		icon = icon.Attr(map[string]string{"height": callerH})
	case callerW != "":
		icon = icon.Width(callerW)
	case callerH != "":
		icon = icon.Height(callerH)
	default:
		icon = icon.Width("16")
	}

	if spriteMode.Load() {
		id := "i-" + spriteID
		sprites.Register(id, rawBody, viewBox)

		var titlePrefix string
		if titleText != "" {
			titlePrefix = "<title>" + html.EscapeString(titleText) + "</title>"
		}
		useBody := titlePrefix + `<use href="#` + id + `"></use>`
		wrapper := swarmicons.New(useBody, icon.Attributes())
		return wrapper.ToHTML()
	}

	return icon.ToHTML()
}

func buildARIAAttrs(attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	aria := map[string]string{}
	for k, v := range attrs {
		switch k {
		case "aria-label", "aria-labelledby":
			aria[k] = v
		case "role":
			aria[k] = v
		case "title":
			if aria["role"] == "" {
				aria["role"] = "img"
			}
		}
	}
	if len(aria) == 0 {
		return nil
	}
	return aria
}

func iconViewBox(icon *swarmicons.Icon) string {
	attrs := icon.Attributes()
	if vb := attrs["viewBox"]; vb != "" {
		return vb
	}
	w, h := attrs["width"], attrs["height"]
	if w != "" && h != "" {
		return "0 0 " + w + " " + h
	}
	return "0 0 24 24"
}

// lookupIcon resolves a "set:name"/bare name to an Icon and sprite ID.
func lookupIcon(name string, callerAttrs ...map[string]string) (*swarmicons.Icon, string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", false
	}

	bare := !strings.Contains(name, ":")

	if !bare {
		prefix := name[:strings.Index(name, ":")]
		if canonical, ok := setAliases[prefix]; ok {
			name = canonical + name[strings.Index(name, ":"):]
		}
	}

	var ca map[string]string
	if len(callerAttrs) > 0 {
		ca = callerAttrs[0]
	}

	if bare {
		if icon, err := manager.Get("local:"+name, ca); err == nil && icon != nil && !icon.IsEmpty() {
			return icon, "local-" + name, true
		}
	}

	icon, err := manager.Get(name, ca)
	if err == nil && icon != nil && !icon.IsEmpty() {
		prefix, iconName := parseIconID(name)
		recordUsed(prefix)
		return icon, prefix + "-" + iconName, true
	}

	if bare && defaultPrefix == "lucide" {
		if alias, ok := aliasMap[name]; ok && alias != name {
			if icon, err := manager.Get(alias, ca); err == nil && icon != nil && !icon.IsEmpty() {
				recordUsed("lucide")
				return icon, "lucide-" + alias, true
			}
		}
	}

	return nil, "", false
}

func parseIconID(id string) (string, string) {
	if idx := strings.Index(id, ":"); idx > 0 {
		return id[:idx], id[idx+1:]
	}
	return defaultPrefix, id
}

func recordUsed(prefix string) {
	usedSets.Store(prefix, struct{}{})
}

// ---------------------------------------------------------------------------
// Dynamic provider registration (called from build init)
// ---------------------------------------------------------------------------

// LoadCollection loads an additional Iconify JSON collection from raw bytes
// and registers it as a provider under its prefix.
func LoadCollection(data []byte) error {
	var meta struct {
		Prefix string    `json:"prefix"`
		Info   *iconInfo `json:"info,omitempty"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}
	if meta.Prefix == "" {
		return fmt.Errorf("icon collection missing prefix")
	}

	provider := swarmicons.NewJsonCollectionFromBytes(data)
	manager.Register(meta.Prefix, provider)

	collectionsMu.Lock()
	collections[meta.Prefix] = &iconifyCollection{Prefix: meta.Prefix, Info: meta.Info}
	collectionsMu.Unlock()
	return nil
}

// LoadIconDirectory registers a DirectoryProvider that loads SVG files from
// dirPath. Bare icon names resolve from this directory FIRST (project overrides win).
// A missing directory is not an error.
func LoadIconDirectory(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil
	}
	provider, err := swarmicons.NewDirectoryProvider(dirPath, swarmicons.WithRecursive(false))
	if err != nil {
		return err
	}

	managerMu.Lock()
	defer managerMu.Unlock()
	manager.Register("local", provider)
	return nil
}

// ---------------------------------------------------------------------------
// License reporting
// ---------------------------------------------------------------------------

// SetLicense is a loaded icon set's license metadata, for attribution.
type SetLicense struct {
	Prefix string
	Title  string
	SPDX   string
	URL    string
}

// LoadedSetLicenses returns license metadata for every loaded collection.
func LoadedSetLicenses() []SetLicense {
	collectionsMu.RLock()
	defer collectionsMu.RUnlock()
	out := make([]SetLicense, 0, len(collections))
	for prefix, col := range collections {
		sl := SetLicense{Prefix: prefix}
		if col.Info != nil && col.Info.License != nil {
			sl.Title = col.Info.License.Title
			sl.SPDX = col.Info.License.SPDX
			sl.URL = col.Info.License.URL
		}
		out = append(out, sl)
	}
	return sortLicenses(out)
}

// UsedSetLicenses returns license metadata for the collections actually
// referenced during the build.
func UsedSetLicenses() []SetLicense {
	collectionsMu.RLock()
	defer collectionsMu.RUnlock()
	var out []SetLicense
	usedSets.Range(func(k, _ any) bool {
		prefix, _ := k.(string)
		if col, ok := collections[prefix]; ok {
			sl := SetLicense{Prefix: prefix}
			if col.Info != nil && col.Info.License != nil {
				sl.Title = col.Info.License.Title
				sl.SPDX = col.Info.License.SPDX
				sl.URL = col.Info.License.URL
			}
			out = append(out, sl)
		}
		return true
	})
	return sortLicenses(out)
}

func sortLicenses(s []SetLicense) []SetLicense {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].Prefix < s[j-1].Prefix; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
	return s
}
