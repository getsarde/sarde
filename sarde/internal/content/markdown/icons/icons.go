package icons

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

//go:embed lucide.json
var lucideJSON []byte

//go:embed tabler.json
var tablerJSON []byte

//go:embed simple-icons.json
var simpleiconsJSON []byte

// setAliases maps a friendly set name to the canonical collection prefix it
// resolves to. "brands" is an alias for the Simple Icons logo set, so
// "brands:github" and "simple-icons:github" resolve to the same icon and dedupe
// to one canonical sprite id / one license-attribution row.
var setAliases = map[string]string{
	"brands": "simple-icons",
}

// IconifyCollection represents an Iconify JSON icon collection.
type IconifyCollection struct {
	Prefix  string                      `json:"prefix"`
	Width   int                         `json:"width"`
	Height  int                         `json:"height"`
	Left    *int                        `json:"left,omitempty"`
	Top     *int                        `json:"top,omitempty"`
	Icons   map[string]IconifyIcon      `json:"icons"`
	Aliases map[string]IconifyIconAlias `json:"aliases"`
	Info    *IconInfo                   `json:"info,omitempty"`
}

// IconInfo holds the subset of an Iconify collection's `info` block we need
// (the set name and its license). Other info fields are ignored on unmarshal.
type IconInfo struct {
	Name    string          `json:"name"`
	License *IconifyLicense `json:"license,omitempty"`
}

// IconifyLicense is a set's license metadata, surfaced for attribution.
type IconifyLicense struct {
	Title string `json:"title"`
	SPDX  string `json:"spdx"`
	URL   string `json:"url"`
}

// IconifyIcon represents a single icon in an Iconify collection. Left/Top are
// pointers because 0 is a meaningful origin (absent must differ from explicit
// 0); Width/Height keep value semantics (0 = inherit collection root).
type IconifyIcon struct {
	Body   string `json:"body"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Left   *int   `json:"left,omitempty"`
	Top    *int   `json:"top,omitempty"`
	Rotate int    `json:"rotate,omitempty"`
	HFlip  bool   `json:"hFlip,omitempty"`
	VFlip  bool   `json:"vFlip,omitempty"`
	Hidden bool   `json:"hidden,omitempty"`
}

// IconifyIconAlias maps an alias name to its parent icon. Per the Iconify spec,
// transform props (rotate/hFlip/vFlip) MERGE with the parent's, while geometry
// and body props OVERRIDE the parent's when present (hence the pointers).
type IconifyIconAlias struct {
	Parent string  `json:"parent"`
	Rotate int     `json:"rotate,omitempty"`
	HFlip  bool    `json:"hFlip,omitempty"`
	VFlip  bool    `json:"vFlip,omitempty"`
	Left   *int    `json:"left,omitempty"`
	Top    *int    `json:"top,omitempty"`
	Width  *int    `json:"width,omitempty"`
	Height *int    `json:"height,omitempty"`
	Body   *string `json:"body,omitempty"`
}

// resolvedIcon is a fully merged icon (alias chain collapsed, dims defaulted)
// ready for iconToSVG. Left/Top are concrete (defaulted to 0).
type resolvedIcon struct {
	Body   string
	Width  int
	Height int
	Left   int
	Top    int
	Rotate int
	HFlip  bool
	VFlip  bool
	Hidden bool
}

// localIcon is an SVG loaded from the project's local icons/ directory.
type localIcon struct {
	Body    string
	ViewBox string
	Width   int
	Height  int
}

// resolver holds loaded icon collections and local-directory icons. The
// embedded sets are bootstrapped once via `once`; all map access is guarded by
// `mu` because LoadCollection / LoadIconDirectory may register sources after
// the first render.
var resolver struct {
	once          sync.Once
	mu            sync.RWMutex
	collections   map[string]*IconifyCollection
	localIcons    map[string]localIcon
	defaultPrefix string
}

// usedSets records the prefixes of collections actually resolved during a
// build, for license-attribution reporting. Written during parallel render.
var usedSets sync.Map

// spriteMode is true when icons render as <use> references into a per-page
// <symbol> sprite (config icons.render: "sprite") rather than full inline
// <svg>. Set once during build init (serial), read lock-free under render.
var spriteMode atomic.Bool

// spriteSymbols holds the <symbol> definition for every icon rendered in sprite
// mode this process, keyed by its full element id ("i-{source}-{name}").
// Append-only and idempotent (icon bodies are deterministic), so it is safe to
// populate concurrently and never needs resetting between builds.
var spriteSymbols sync.Map

// spriteSymbol is a registered sprite <symbol>'s viewBox + id-namespaced body.
type spriteSymbol struct {
	ViewBox string
	Body    string
}

func init() {
	resolver.collections = make(map[string]*IconifyCollection)
	resolver.localIcons = make(map[string]localIcon)
	resolver.defaultPrefix = "lucide"
}

func ensureLoaded() {
	resolver.once.Do(func() {
		resolver.mu.Lock()
		defer resolver.mu.Unlock()
		for _, set := range []struct {
			prefix string
			data   []byte
		}{
			{"lucide", lucideJSON},
			{"tabler", tablerJSON},
			{"simple-icons", simpleiconsJSON},
		} {
			var col IconifyCollection
			if err := json.Unmarshal(set.data, &col); err == nil {
				resolver.collections[set.prefix] = &col
			}
		}
	})
}

// SetDefaultPrefix sets the icon set used for bare (prefixless) names. Empty
// values are ignored so a missing config key keeps the "lucide" default.
func SetDefaultPrefix(p string) {
	if p == "" {
		return
	}
	resolver.mu.Lock()
	resolver.defaultPrefix = p
	resolver.mu.Unlock()
}

// SetRenderMode selects the icon output mode. "sprite" enables per-page
// <symbol> sprite output (Render emits <use> references); anything else
// (including "" and "inline") keeps the default full-<svg>-per-use output.
// Call once during build init, before parallel render.
func SetRenderMode(m string) {
	spriteMode.Store(strings.EqualFold(strings.TrimSpace(m), "sprite"))
}

// SpriteMode reports whether sprite output is enabled.
func SpriteMode() bool {
	return spriteMode.Load()
}

// Get returns an inline SVG for the given icon name.
// Supports "set:name" (e.g. "lucide:rocket") or bare "name" (defaults to the
// configured default prefix). Returns empty string if the icon is not found.
func Get(name string) string {
	return GetWithClass(name, "")
}

// GetWithClass returns an inline SVG with an optional CSS class attribute.
// Used by block extensions (aside/card/linkcard/linkbutton). It performs no
// fallback and emits no accessibility attributes — width/height are hardcoded
// to 16 so the output is intentionally identical to the legacy implementation.
// For the template func and inline Markdown extension use Render instead.
func GetWithClass(name, class string) string {
	ri, _, ok := lookupIcon(name)
	if !ok {
		return ""
	}
	body, viewBox := iconToSVG(ri)
	return renderSVG(class, "16", "16", viewBox, []svgAttr{{"aria-hidden", "true"}, {"focusable", "false"}}, body)
}

// Render returns an inline SVG for the template func and the inline Markdown
// extension. Unlike GetWithClass it falls back to the "circle-help" icon on a
// miss, applies user transforms (rotate/flip/opacity/title), concatenates
// baseClass with a caller-supplied class, derives missing width/height from the
// viewBox aspect ratio, and emits accessibility attributes automatically:
// decorative (aria-hidden + focusable="false") by default, promoted to
// role="img" when an accessible name (aria-label / aria-labelledby / title) is
// supplied. Attribute values are HTML-escaped and unsafe attribute keys are
// dropped, so it is safe to wrap the result in template.HTML.
func Render(name, baseClass string, attrs map[string]string) string {
	ri, idBase, ok := lookupIcon(name)
	if !ok {
		ri, idBase, ok = lookupIcon("circle-help")
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

	// Bake the icon's intrinsic transforms into the body + viewBox, then derive
	// the emitted width/height from the (possibly swapped) viewBox.
	body, viewBox := iconToSVG(ri)
	boxW, boxH := boxDimsFromViewBox(viewBox)
	width, height := calculateSize(callerW, callerH, boxW, boxH)

	// Compose the user-transform style (rotate/flip → CSS transform), merged
	// with any caller style. This layer is independent of the baked intrinsic
	// transform above: intrinsic = inside the SVG body, user = CSS on the box.
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

	// Accessibility: a <title> also provides an accessible name. It is emitted
	// as a prefix on the outer <svg> body — kept out of the shared <symbol>,
	// which carries no per-instance name.
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

	// Sprite mode: register the icon's geometry as a reusable <symbol> and
	// reference it with <use>. The outer <svg> keeps the exact same class /
	// dims / style / ARIA as the inline path; only the body differs.
	if spriteMode.Load() {
		registerSymbol(idBase, body, viewBox)
		return renderSVG(class, width, height, viewBox, extras, titlePrefix+`<use href="#i-`+idBase+`"></use>`)
	}

	return renderSVG(class, width, height, viewBox, extras, titlePrefix+body)
}

// lookupIcon resolves a "set:name"/bare name to a fully merged icon. For bare
// names the local icons/ directory is consulted FIRST (project overrides win),
// then the default-prefix collection; "set:name" always routes to that set.
// The returned idBase ("{source}-{name}") identifies the resolved icon for the
// sprite element id ("i-"+idBase): "local" for local-dir icons, else the set
// prefix; name is the canonical resolved name (post Lucide-shorthand alias) so
// equivalent references dedupe to one <symbol>.
func lookupIcon(name string) (resolvedIcon, string, bool) {
	ensureLoaded()

	bare := !strings.Contains(name, ":")
	setName, iconName := parseIconID(name)
	// Canonicalize set aliases (e.g. "brands" -> "simple-icons") so the alias and
	// its canonical name share one collection, one sprite id, and one license row.
	if c, ok := setAliases[setName]; ok {
		setName = c
	}

	// 1. Local directory (bare names only).
	if bare {
		resolver.mu.RLock()
		loc, ok := resolver.localIcons[iconName]
		resolver.mu.RUnlock()
		if ok {
			return localToResolved(loc), "local-" + iconName, true
		}
	}

	// 2. Iconify collection.
	resolver.mu.RLock()
	col, found := resolver.collections[setName]
	resolver.mu.RUnlock()
	if !found {
		return resolvedIcon{}, "", false
	}

	if ri, ok := resolveIcon(col, iconName); ok {
		recordUsed(setName)
		return ri, setName + "-" + iconName, true
	}
	// 3. Lucide shorthand aliases (e.g. "warning" -> "alert-triangle"). These
	// are Lucide-specific, so only apply them when resolving the lucide set.
	if setName == "lucide" {
		if alias := resolveAlias(iconName); alias != iconName {
			if ri, ok := resolveIcon(col, alias); ok {
				recordUsed(setName)
				return ri, setName + "-" + alias, true
			}
		}
	}
	return resolvedIcon{}, "", false
}

// resolveIcon walks the alias chain for iconName within col, merging Iconify
// transforms (rotate additive mod 4; hFlip/vFlip XOR) and applying geometry/
// body overrides (nearest alias to the requested name wins). Collection-root
// dimensions fill in where the icon supplies none. Returns ok=false if the icon
// cannot be resolved within 10 hops.
func resolveIcon(col *IconifyCollection, iconName string) (resolvedIcon, bool) {
	var accRotate int
	var accHFlip, accVFlip bool
	// Geometry/body overrides from aliases: first non-nil in the chain wins.
	var ovLeft, ovTop, ovWidth, ovHeight *int
	var ovBody *string

	current := iconName
	for i := 0; i < 10; i++ {
		if icon, ok := col.Icons[current]; ok {
			ri := resolvedIcon{
				Body:   icon.Body,
				Width:  icon.Width,
				Height: icon.Height,
				Left:   derefOr(icon.Left, 0),
				Top:    derefOr(icon.Top, 0),
				Rotate: icon.Rotate,
				HFlip:  icon.HFlip,
				VFlip:  icon.VFlip,
				Hidden: icon.Hidden,
			}

			// Apply alias overrides (geometry/body) — nearest alias wins.
			if ovBody != nil {
				ri.Body = *ovBody
			}
			if ovWidth != nil {
				ri.Width = *ovWidth
			}
			if ovHeight != nil {
				ri.Height = *ovHeight
			}
			if ovLeft != nil {
				ri.Left = *ovLeft
			}
			if ovTop != nil {
				ri.Top = *ovTop
			}

			// Merge accumulated transforms from the alias chain.
			ri.Rotate = ((ri.Rotate+accRotate)%4 + 4) % 4
			ri.HFlip = ri.HFlip != accHFlip
			ri.VFlip = ri.VFlip != accVFlip

			// Collection-root defaults where the icon (and aliases) gave none.
			if ri.Width == 0 {
				ri.Width = col.Width
			}
			if ri.Height == 0 {
				ri.Height = col.Height
			}
			if ri.Width == 0 {
				ri.Width = 24
			}
			if ri.Height == 0 {
				ri.Height = 24
			}
			if icon.Left == nil && ovLeft == nil && col.Left != nil {
				ri.Left = *col.Left
			}
			if icon.Top == nil && ovTop == nil && col.Top != nil {
				ri.Top = *col.Top
			}
			return ri, true
		}

		alias, ok := col.Aliases[current]
		if !ok {
			return resolvedIcon{}, false
		}
		accRotate += alias.Rotate
		accHFlip = accHFlip != alias.HFlip
		accVFlip = accVFlip != alias.VFlip
		if ovBody == nil && alias.Body != nil {
			ovBody = alias.Body
		}
		if ovWidth == nil && alias.Width != nil {
			ovWidth = alias.Width
		}
		if ovHeight == nil && alias.Height != nil {
			ovHeight = alias.Height
		}
		if ovLeft == nil && alias.Left != nil {
			ovLeft = alias.Left
		}
		if ovTop == nil && alias.Top != nil {
			ovTop = alias.Top
		}
		current = alias.Parent
	}
	return resolvedIcon{}, false
}

// localToResolved turns a local-directory SVG into a resolvedIcon, deriving its
// box from the viewBox (falling back to width/height attrs, then 24).
func localToResolved(loc localIcon) resolvedIcon {
	minX, minY, w, h := parseViewBox(loc.ViewBox)
	if w == 0 {
		w = loc.Width
	}
	if h == 0 {
		h = loc.Height
	}
	if w == 0 {
		w = 24
	}
	if h == 0 {
		h = 24
	}
	return resolvedIcon{Body: loc.Body, Width: w, Height: h, Left: minX, Top: minY}
}

// iconToSVG bakes the icon's intrinsic left/top/rotate/hFlip/vFlip into a
// <g transform="..."> wrapper and returns the (possibly wrapped) body plus the
// output viewBox. A port of Iconify's iconToSVG (packages/utils/src/svg/build):
// flips are appended, the rotate transform is prepended, and odd rotations
// (90/270) swap the box width/height. Icons with no transforms (e.g. all of
// Lucide) get no <g> wrapper, so block-extension output stays byte-identical.
func iconToSVG(ri resolvedIcon) (string, string) {
	left, top := ri.Left, ri.Top
	w, h := ri.Width, ri.Height
	body := ri.Body

	rotate := ((ri.Rotate % 4) + 4) % 4
	hFlip, vFlip := ri.HFlip, ri.VFlip
	if hFlip && vFlip {
		hFlip, vFlip = false, false
		rotate = (rotate + 2) % 4
	}

	var parts []string
	if hFlip {
		parts = append(parts, fmt.Sprintf("translate(%d %d)", w+left, -top), "scale(-1 1)")
		left, top = 0, 0
	} else if vFlip {
		parts = append(parts, fmt.Sprintf("translate(%d %d)", -left, h+top), "scale(1 -1)")
		left, top = 0, 0
	}

	switch rotate {
	case 1:
		t := fmtNum(float64(h)/2 + float64(top))
		parts = append([]string{fmt.Sprintf("rotate(90 %s %s)", t, t)}, parts...)
	case 2:
		cx := fmtNum(float64(w)/2 + float64(left))
		cy := fmtNum(float64(h)/2 + float64(top))
		parts = append([]string{fmt.Sprintf("rotate(180 %s %s)", cx, cy)}, parts...)
	case 3:
		t := fmtNum(float64(w)/2 + float64(left))
		parts = append([]string{fmt.Sprintf("rotate(-90 %s %s)", t, t)}, parts...)
	}

	if rotate%2 == 1 {
		if left != top {
			left, top = top, left
		}
		if w != h {
			w, h = h, w
		}
	}

	if len(parts) > 0 {
		body = `<g transform="` + strings.Join(parts, " ") + `">` + body + `</g>`
	}

	return body, fmt.Sprintf("%d %d %d %d", left, top, w, h)
}

// calculateSize resolves the emitted width/height. Port of Iconify's
// calculateSize behavior adapted to Sarde's CSS-class sizing default:
//   - both empty            → "16","16" (the legacy default; keeps callers identical)
//   - both supplied         → pass through unchanged
//   - only one supplied     → derive the other from the viewBox aspect ratio
//   - "unset"/"none"/"undefined" → omit that attribute (let CSS size it)
//   - "auto"                → use the numeric viewBox dimension
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
	sizeMissing sizeState = iota // caller gave nothing
	sizeOmit                     // caller asked to drop the attribute
	sizeValue                    // caller gave a concrete value
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

// deriveSize scales a size string by ratio, preserving a trailing unit (e.g.
// "1em" × 1.5 → "1.5em"). Mirrors Iconify: ceil(value * ratio * 100) / 100.
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

// splitNumUnit splits a leading numeric prefix from a trailing unit.
func splitNumUnit(s string) (num, unit string) {
	i := 0
	for i < len(s) && (s[i] == '-' || s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	return s[:i], s[i:]
}

// fmtNum formats a float without a trailing ".0" (12.0 → "12", 12.5 → "12.5").
func fmtNum(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// parseViewBox parses a "minX minY width height" viewBox (float-tolerant,
// truncated to ints). Returns all zeros if the string is not a 4-number box.
func parseViewBox(vb string) (minX, minY, w, h int) {
	var fx, fy, fw, fh float64
	if n, _ := fmt.Sscanf(strings.TrimSpace(vb), "%g %g %g %g", &fx, &fy, &fw, &fh); n == 4 {
		return int(fx), int(fy), int(fw), int(fh)
	}
	return 0, 0, 0, 0
}

// boxDimsFromViewBox extracts just the width/height from a viewBox string.
func boxDimsFromViewBox(vb string) (w, h int) {
	_, _, w, h = parseViewBox(vb)
	return w, h
}

var (
	reID      = regexp.MustCompile(`(^|\s)id="([^"]+)"`)
	reURLRef  = regexp.MustCompile(`url\(#([^)]+)\)`)
	reHrefRef = regexp.MustCompile(`((?:xlink:)?href)="#([^"]+)"`)
)

// replaceIDs rewrites internal id/url(#…)/href="#…" references in an SVG body
// by appending suffix, so multiple instances of an icon using <mask>/<clipPath>/
// gradients on one page don't collide. suffix must be unique per emitted SVG.
// NOTE: not applied by the Phase-2 inline render path (a deterministic suffix
// under parallel render would be fragile, and the bundled outline sets carry no
// internal ids); it exists for the Phase-3 <symbol> sprite mode.
func replaceIDs(body, suffix string) string {
	if suffix == "" {
		return body
	}
	body = reID.ReplaceAllString(body, `${1}id="${2}-`+suffix+`"`)
	body = reURLRef.ReplaceAllString(body, `url(#${1}-`+suffix+`)`)
	body = reHrefRef.ReplaceAllString(body, `${1}="#${2}-`+suffix+`"`)
	return body
}

// registerSymbol stores an icon's geometry as a sprite <symbol> definition,
// keyed by its element id ("i-"+idBase). Internal ids in the body are namespaced
// with idBase (via replaceIDs) so repeated symbols on a page can't collide.
// Idempotent: a given id is only computed and stored once per process.
func registerSymbol(idBase, body, viewBox string) {
	id := "i-" + idBase
	if _, ok := spriteSymbols.Load(id); ok {
		return
	}
	spriteSymbols.LoadOrStore(id, spriteSymbol{ViewBox: viewBox, Body: replaceIDs(body, idBase)})
}

// reSpriteUse matches a sprite <use> reference (href="#i-…" or its xlink form)
// so SpriteForHTML can discover which symbols a finished page needs.
var reSpriteUse = regexp.MustCompile(`(?:xlink:)?href="#(i-[a-zA-Z0-9_-]+)"`)

// SpriteForHTML scans finished page HTML for sprite <use> references and returns
// a hidden <svg> holding one <symbol> per unique referenced icon, in sorted
// (deterministic) order. Returns nil when sprite mode is off or the page
// references no registered sprite icons. Unknown ids (e.g. an author's own
// "#i-…" anchor) are skipped via the spriteSymbols lookup.
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

// svgAttr is a single pre-escaped SVG attribute.
type svgAttr struct{ key, val string }

// renderSVG assembles the final <svg> markup. class is emitted first (matching
// the legacy block-extension output); extras are emitted in sorted key order
// after viewBox so output is deterministic regardless of map iteration order.
// An empty width/height is omitted (so calculateSize can defer to CSS sizing).
// extras values must already be escaped by the caller; class/width/height are
// escaped here.
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

// buildTransform turns rotate/flip attributes into a CSS transform declaration
// (e.g. `transform: rotate(90deg) scaleX(-1)`). Numeric rotate values get a
// `deg` unit; flip accepts horizontal/vertical/both (and h/v shorthands).
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

// isNumericValue reports whether s is a plain (optionally negative/decimal) number.
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

// validAttrKey restricts pass-through attribute keys to a safe character set.
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

// derefOr returns *p, or def when p is nil.
func derefOr(p *int, def int) int {
	if p != nil {
		return *p
	}
	return def
}

// parseIconID splits "set:name" into (set, name). Bare "name" uses the
// configured default prefix.
func parseIconID(id string) (string, string) {
	if idx := strings.Index(id, ":"); idx > 0 {
		return id[:idx], id[idx+1:]
	}
	resolver.mu.RLock()
	prefix := resolver.defaultPrefix
	resolver.mu.RUnlock()
	if prefix == "" {
		prefix = "lucide"
	}
	return prefix, id
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

func resolveAlias(name string) string {
	if alias, ok := aliasMap[name]; ok {
		return alias
	}
	return name
}

// recordUsed marks a collection prefix as referenced during the build.
func recordUsed(prefix string) {
	usedSets.Store(prefix, struct{}{})
}

// SetLicense is a loaded icon set's license metadata, for attribution.
type SetLicense struct {
	Prefix string
	Title  string
	SPDX   string
	URL    string
}

func collectionLicense(prefix string, col *IconifyCollection) SetLicense {
	sl := SetLicense{Prefix: prefix}
	if col.Info != nil && col.Info.License != nil {
		sl.Title = col.Info.License.Title
		sl.SPDX = col.Info.License.SPDX
		sl.URL = col.Info.License.URL
	}
	return sl
}

// LoadedSetLicenses returns license metadata for every loaded collection,
// sorted by prefix. Exposed to templates (e.g. for an attribution/credits page)
// — it lists all loaded sets and so is available at render time.
func LoadedSetLicenses() []SetLicense {
	ensureLoaded()
	resolver.mu.RLock()
	defer resolver.mu.RUnlock()
	out := make([]SetLicense, 0, len(resolver.collections))
	for prefix, col := range resolver.collections {
		out = append(out, collectionLicense(prefix, col))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Prefix < out[j].Prefix })
	return out
}

// UsedSetLicenses returns license metadata for the collections actually
// referenced during the build, sorted by prefix. Used for the build-end
// attribution warning (precise: only sets whose icons were rendered).
func UsedSetLicenses() []SetLicense {
	resolver.mu.RLock()
	defer resolver.mu.RUnlock()
	var out []SetLicense
	usedSets.Range(func(k, _ any) bool {
		prefix, _ := k.(string)
		if col, ok := resolver.collections[prefix]; ok {
			out = append(out, collectionLicense(prefix, col))
		}
		return true
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Prefix < out[j].Prefix })
	return out
}

// LoadCollection loads an additional Iconify JSON collection from raw JSON bytes
// and registers it under its prefix.
func LoadCollection(data []byte) error {
	var col IconifyCollection
	if err := json.Unmarshal(data, &col); err != nil {
		return err
	}
	if col.Prefix == "" {
		return fmt.Errorf("icon collection missing prefix")
	}
	ensureLoaded()
	resolver.mu.Lock()
	resolver.collections[col.Prefix] = &col
	resolver.mu.Unlock()
	return nil
}
