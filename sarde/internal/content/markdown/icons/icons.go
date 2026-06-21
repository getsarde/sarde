package icons

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	swarmicons "github.com/frostybee/go-swarm-icons"
	"github.com/frostybee/go-swarm-icons/lucide"
)

// manager is the central icon lookup engine backed by go-swarm-icons.
var manager *swarmicons.IconManager

// managerMu guards lazy initialization and dynamic provider registration.
var managerMu sync.Mutex

// defaultPrefix is the prefix used for bare (prefixless) icon names.
var defaultPrefix = "lucide"

// setAliases maps a friendly set name to the canonical collection prefix it
// resolves to. "brands" is an alias for the Simple Icons logo set.
var setAliases = map[string]string{
	"brands": "simple-icons",
}

// aliasMap maps common shorthand names to their Lucide equivalents.
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

// usedSets records the prefixes of collections actually resolved during a
// build, for license-attribution reporting.
var usedSets sync.Map

// spriteMode is true when icons render as <use> references into a per-page
// <symbol> sprite rather than full inline <svg>.
var spriteMode atomic.Bool

// spriteSymbols holds <symbol> definitions keyed by element id ("i-{source}-{name}").
var spriteSymbols sync.Map

type spriteSymbol struct {
	ViewBox string
	Body    string
}

// collections stores loaded Iconify JSON data for license reporting.
// Access guarded by collectionsMu.
var (
	collectionsMu sync.RWMutex
	collections   = make(map[string]*iconifyCollection)
)

type iconifyCollection struct {
	Prefix string       `json:"prefix"`
	Info   *iconInfo    `json:"info,omitempty"`
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
	manager = swarmicons.Default(lucide.Provider())
	manager.SetIgnoreNotFound(true)

	// Register embedded tabler and simple-icons sets.
	registerEmbeddedSets()
}

//go:embed tabler.json
var tablerJSON []byte

//go:embed simple-icons.json
var simpleiconsJSON []byte

func registerEmbeddedSets() {
	for _, set := range []struct {
		prefix string
		data   []byte
	}{
		{"lucide", lucideJSON},
		{"tabler", tablerJSON},
		{"simple-icons", simpleiconsJSON},
	} {
		if set.prefix != "lucide" {
			provider := swarmicons.NewJsonCollectionFromBytes(set.data)
			manager.Register(set.prefix, provider)
		}
		extractLicenseInfo(set.prefix, set.data)
	}
}

//go:embed lucide.json
var lucideJSON []byte

func extractLicenseInfo(prefix string, data []byte) {
	var col iconifyCollection
	if err := json.Unmarshal(data, &col); err == nil {
		col.Prefix = prefix
		collectionsMu.Lock()
		collections[prefix] = &col
		collectionsMu.Unlock()
	}
}

// Manager returns the underlying IconManager for use by the Goldmark extension.
func Manager() *swarmicons.IconManager {
	return manager
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

// Get returns an inline SVG for the given icon name.
func Get(name string) string {
	return GetWithClass(name, "")
}

// GetWithClass returns an inline SVG with an optional CSS class attribute.
// Used by block extensions (aside/card/linkcard/linkbutton). No fallback;
// width/height hardcoded to 16.
func GetWithClass(name, class string) string {
	resolved, idBase, ok := lookupIcon(name)
	if !ok {
		return ""
	}
	_ = idBase
	return renderSVG(class, "16", "16", resolved.viewBox, []svgAttr{{"aria-hidden", "true"}, {"focusable", "false"}}, resolved.body)
}

// Render returns an inline SVG for the template func and inline Markdown extension.
// Falls back to "circle-help" on miss, applies transforms, accessibility attributes.
func Render(name, baseClass string, attrs map[string]string) string {
	resolved, idBase, ok := lookupIcon(name)
	if !ok {
		resolved, idBase, ok = lookupIcon("circle-help")
		if !ok {
			return ""
		}
	}

	class := baseClass
	var callerW, callerH string
	var rotate, flip, userStyle, titleText string
	var extras []svgAttr
	hasName := false
	roleSet := false

	for k, v := range attrs {
		switch k {
		case "class":
			if v != "" {
				if class != "" {
					class += " " + v
				} else {
					class = v
				}
			}
		case "width":
			callerW = v
		case "height":
			callerH = v
		case "rotate":
			rotate = v
		case "flip":
			flip = v
		case "style":
			userStyle = v
		case "title":
			titleText = v
		case "role":
			roleSet = true
			hasName = true
			extras = append(extras, svgAttr{"role", html.EscapeString(v)})
		case "aria-label", "aria-labelledby":
			hasName = true
			extras = append(extras, svgAttr{k, html.EscapeString(v)})
		default:
			if validAttrKey(k) {
				extras = append(extras, svgAttr{k, html.EscapeString(v)})
			}
		}
	}

	body := resolved.body
	viewBox := resolved.viewBox
	boxW, boxH := boxDimsFromViewBox(viewBox)
	width, height := calculateSize(callerW, callerH, boxW, boxH)

	style := buildTransform(rotate, flip)
	if userStyle != "" {
		if style != "" {
			style += "; " + userStyle
		} else {
			style = userStyle
		}
	}
	if style != "" {
		extras = append(extras, svgAttr{"style", html.EscapeString(style)})
	}

	var titlePrefix string
	if titleText != "" {
		hasName = true
		titlePrefix = "<title>" + html.EscapeString(titleText) + "</title>"
	}
	if hasName {
		if !roleSet {
			extras = append(extras, svgAttr{"role", "img"})
		}
	} else {
		extras = append(extras, svgAttr{"aria-hidden", "true"}, svgAttr{"focusable", "false"})
	}

	if spriteMode.Load() {
		registerSymbol(idBase, body, viewBox)
		return renderSVG(class, width, height, viewBox, extras, titlePrefix+`<use href="#i-`+idBase+`"></use>`)
	}

	return renderSVG(class, width, height, viewBox, extras, titlePrefix+body)
}

// resolvedIconData holds the content and geometry extracted from an icon lookup.
type resolvedIconData struct {
	body    string
	viewBox string
}

// lookupIcon resolves a "set:name"/bare name to icon content and viewBox.
// For bare names the "local" provider is consulted FIRST (project overrides win),
// then the default-prefix collection. "set:name" always routes to that set.
func lookupIcon(name string) (resolvedIconData, string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return resolvedIconData{}, "", false
	}

	bare := !strings.Contains(name, ":")

	// Resolve set aliases in the prefix position (e.g. "brands:github" → "simple-icons:github").
	if !bare {
		prefix := name[:strings.Index(name, ":")]
		if canonical, ok := setAliases[prefix]; ok {
			name = canonical + name[strings.Index(name, ":"):]
		}
	}

	// 1. Local directory (bare names only) — project overrides win.
	if bare {
		if icon, err := manager.Get("local:" + name); err == nil && icon != nil && !icon.IsEmpty() {
			data := extractIconData(icon)
			return data, "local-" + name, true
		}
	}

	// 2. Try the name directly via the manager.
	icon, err := manager.Get(name)
	if err == nil && icon != nil && !icon.IsEmpty() {
		prefix, iconName := parseIconID(name)
		recordUsed(prefix)
		data := extractIconData(icon)
		return data, prefix + "-" + iconName, true
	}

	// 3. Lucide shorthand aliases (bare names only, gated to lucide default).
	if bare && defaultPrefix == "lucide" {
		if alias, ok := aliasMap[name]; ok && alias != name {
			if icon, err := manager.Get(alias); err == nil && icon != nil && !icon.IsEmpty() {
				recordUsed("lucide")
				data := extractIconData(icon)
				return data, "lucide-" + alias, true
			}
		}
	}

	return resolvedIconData{}, "", false
}

func extractIconData(icon *swarmicons.Icon) resolvedIconData {
	content := icon.Content()
	attrs := icon.Attributes()
	viewBox := attrs["viewBox"]
	if viewBox == "" {
		w := attrs["width"]
		h := attrs["height"]
		if w != "" && h != "" {
			viewBox = "0 0 " + w + " " + h
		} else {
			viewBox = "0 0 24 24"
		}
	}
	return resolvedIconData{body: content, viewBox: viewBox}
}

// parseIconID splits "set:name" into (set, name). Bare "name" uses the default prefix.
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
// Sprite mode
// ---------------------------------------------------------------------------

var (
	reID      = regexp.MustCompile(`(^|\s)id="([^"]+)"`)
	reURLRef  = regexp.MustCompile(`url\(#([^)]+)\)`)
	reHrefRef = regexp.MustCompile(`((?:xlink:)?href)="#([^"]+)"`)
)

func replaceIDs(body, suffix string) string {
	if suffix == "" {
		return body
	}
	body = reID.ReplaceAllString(body, `${1}id="${2}-`+suffix+`"`)
	body = reURLRef.ReplaceAllString(body, `url(#${1}-`+suffix+`)`)
	body = reHrefRef.ReplaceAllString(body, `${1}="#${2}-`+suffix+`"`)
	return body
}

func registerSymbol(idBase, body, viewBox string) {
	id := "i-" + idBase
	if _, ok := spriteSymbols.Load(id); ok {
		return
	}
	spriteSymbols.LoadOrStore(id, spriteSymbol{ViewBox: viewBox, Body: replaceIDs(body, idBase)})
}

var reSpriteUse = regexp.MustCompile(`(?:xlink:)?href="#(i-[a-zA-Z0-9_-]+)"`)

// SpriteForHTML scans finished page HTML for sprite <use> references and returns
// a hidden <svg> holding one <symbol> per unique referenced icon.
func SpriteForHTML(pageHTML []byte) []byte {
	if !spriteMode.Load() {
		return nil
	}
	matches := reSpriteUse.FindAllSubmatch(pageHTML, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	var ids []string
	for _, m := range matches {
		id := string(m[1])
		if _, dup := seen[id]; dup {
			continue
		}
		if _, ok := spriteSymbols.Load(id); !ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Strings(ids)

	var b strings.Builder
	b.WriteString(`<svg aria-hidden="true" style="position:absolute;width:0;height:0;overflow:hidden">`)
	for _, id := range ids {
		v, _ := spriteSymbols.Load(id)
		sym := v.(spriteSymbol)
		b.WriteString(`<symbol id="`)
		b.WriteString(id)
		b.WriteString(`" viewBox="`)
		b.WriteString(sym.ViewBox)
		b.WriteString(`">`)
		b.WriteString(sym.Body)
		b.WriteString(`</symbol>`)
	}
	b.WriteString(`</svg>`)
	return []byte(b.String())
}

// ---------------------------------------------------------------------------
// SVG rendering
// ---------------------------------------------------------------------------

type svgAttr struct{ key, val string }

func renderSVG(class, width, height, viewBox string, extras []svgAttr, body string) string {
	var b strings.Builder
	b.WriteString("<svg")
	if class != "" {
		b.WriteString(` class="`)
		b.WriteString(html.EscapeString(class))
		b.WriteString(`"`)
	}
	b.WriteString(` xmlns="http://www.w3.org/2000/svg"`)
	if width != "" {
		b.WriteString(` width="`)
		b.WriteString(html.EscapeString(width))
		b.WriteString(`"`)
	}
	if height != "" {
		b.WriteString(` height="`)
		b.WriteString(html.EscapeString(height))
		b.WriteString(`"`)
	}
	b.WriteString(` viewBox="`)
	b.WriteString(viewBox)
	b.WriteString(`"`)
	if len(extras) > 0 {
		sort.Slice(extras, func(i, j int) bool { return extras[i].key < extras[j].key })
		for _, a := range extras {
			b.WriteString(" ")
			b.WriteString(a.key)
			b.WriteString(`="`)
			b.WriteString(a.val)
			b.WriteString(`"`)
		}
	}
	b.WriteString(">")
	b.WriteString(body)
	b.WriteString("</svg>")
	return b.String()
}

// ---------------------------------------------------------------------------
// Size calculation
// ---------------------------------------------------------------------------

func calculateSize(callerW, callerH string, boxW, boxH int) (string, string) {
	wState, wVal := classifySize(callerW, boxW)
	hState, hVal := classifySize(callerH, boxH)

	switch {
	case wState == sizeMissing && hState == sizeMissing:
		return "16", "16"
	case wState == sizeValue && hState == sizeMissing && boxW != 0:
		return wVal, deriveSize(wVal, float64(boxH)/float64(boxW))
	case hState == sizeValue && wState == sizeMissing && boxH != 0:
		return deriveSize(hVal, float64(boxW)/float64(boxH)), hVal
	default:
		outW, outH := wVal, hVal
		if wState != sizeValue {
			outW = ""
		}
		if hState != sizeValue {
			outH = ""
		}
		return outW, outH
	}
}

type sizeState int

const (
	sizeMissing sizeState = iota
	sizeOmit
	sizeValue
)

func classifySize(s string, boxDim int) (sizeState, string) {
	if s == "" {
		return sizeMissing, ""
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "unset", "none", "undefined":
		return sizeOmit, ""
	case "auto":
		return sizeValue, strconv.Itoa(boxDim)
	}
	return sizeValue, s
}

func deriveSize(size string, ratio float64) string {
	num, unit := splitNumUnit(size)
	if num == "" {
		return size
	}
	f, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return size
	}
	return fmtNum(math.Ceil(f*ratio*100)/100) + unit
}

func splitNumUnit(s string) (num, unit string) {
	i := 0
	for i < len(s) && (s[i] == '-' || s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	return s[:i], s[i:]
}

func fmtNum(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func parseViewBox(vb string) (minX, minY, w, h int) {
	var fx, fy, fw, fh float64
	if n, _ := fmt.Sscanf(strings.TrimSpace(vb), "%g %g %g %g", &fx, &fy, &fw, &fh); n == 4 {
		return int(fx), int(fy), int(fw), int(fh)
	}
	return 0, 0, 0, 0
}

func boxDimsFromViewBox(vb string) (w, h int) {
	_, _, w, h = parseViewBox(vb)
	return w, h
}

// ---------------------------------------------------------------------------
// CSS transforms
// ---------------------------------------------------------------------------

func buildTransform(rotate, flip string) string {
	var parts []string
	if r := strings.TrimSpace(rotate); r != "" {
		if isNumericValue(r) {
			parts = append(parts, "rotate("+r+"deg)")
		} else {
			parts = append(parts, "rotate("+r+")")
		}
	}
	switch strings.ToLower(strings.TrimSpace(flip)) {
	case "horizontal", "h":
		parts = append(parts, "scaleX(-1)")
	case "vertical", "v":
		parts = append(parts, "scaleY(-1)")
	case "horizontal,vertical", "vertical,horizontal", "both":
		parts = append(parts, "scaleX(-1)", "scaleY(-1)")
	}
	if len(parts) == 0 {
		return ""
	}
	return "transform: " + strings.Join(parts, " ")
}

func isNumericValue(s string) bool {
	if s == "" {
		return false
	}
	dot := false
	for i, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r == '-' && i == 0:
		case r == '.' && !dot:
			dot = true
		default:
			return false
		}
	}
	return true
}

func validAttrKey(k string) bool {
	if k == "" {
		return false
	}
	for _, r := range k {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == ':' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Dynamic provider registration (called from build init)
// ---------------------------------------------------------------------------

// LoadCollection loads an additional Iconify JSON collection from raw bytes
// and registers it as a provider under its prefix.
func LoadCollection(data []byte) error {
	var meta struct {
		Prefix string       `json:"prefix"`
		Info   *iconInfo    `json:"info,omitempty"`
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

	// Register as a chain: local dir checked first, then current lucide provider.
	// We use "local" prefix internally but access via the default prefix.
	// Actually, we register a chain provider under the default prefix that checks
	// the directory first, then the existing provider.
	managerMu.Lock()
	defer managerMu.Unlock()

	// Create a chain that checks local dir first, then falls back to the existing
	// default prefix provider.
	// Since the manager's Get already handles prefix routing, we register the
	// directory under a "local" prefix and handle fallback in lookupIcon.
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
	sort.Slice(out, func(i, j int) bool { return out[i].Prefix < out[j].Prefix })
	return out
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
	sort.Slice(out, func(i, j int) bool { return out[i].Prefix < out[j].Prefix })
	return out
}
