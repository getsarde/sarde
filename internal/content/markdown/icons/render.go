package icons

import (
	"html"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	swarmicons "github.com/frostybee/go-swarm-icons"
)

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

	icon = applyIconAttrs(icon, attrs, callerW, callerH, userStyle)

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

// applyIconAttrs applies the rotate, flip, style, and width/height overrides
// to icon and returns the updated icon. Extracted from Render to keep the
// transform/sizing logic isolated from attribute classification.
func applyIconAttrs(icon *swarmicons.Icon, attrs map[string]string, callerW, callerH, userStyle string) *swarmicons.Icon {
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

	return icon
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
