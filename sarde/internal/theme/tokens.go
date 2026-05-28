package theme

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// DeriveTokens auto-generates variant tokens from the accent color.
// When accent is a valid hex color, it derives:
//   - accent-hover: 10% darker
//   - accent-high: 20% lighter (for dark mode emphasis)
//   - accent-low: rgba() at 10% opacity (for subtle backgrounds)
//
// Derivation is skipped if a key already exists (user overrides win).
// Non-hex accent values are skipped gracefully.
func DeriveTokens(tokens map[string]string) map[string]string {
	primary, ok := tokens["accent"]
	if !ok || primary == "" {
		return tokens
	}

	// Try hex path first.
	if r, g, b, err := parseHex(primary); err == nil {
		h, s, l := rgbToHSL(r, g, b)

		if _, exists := tokens["accent-hover"]; !exists {
			hr, hg, hb := hslToRGB(h, s, clamp(l-0.10, 0, 1))
			tokens["accent-hover"] = toHex(hr, hg, hb)
		}
		if _, exists := tokens["accent-high"]; !exists {
			hr, hg, hb := hslToRGB(h, s, clamp(l+0.20, 0, 1))
			tokens["accent-high"] = toHex(hr, hg, hb)
		}
		if _, exists := tokens["accent-low"]; !exists {
			tokens["accent-low"] = fmt.Sprintf("rgba(%d, %d, %d, 0.1)", r, g, b)
		}
		return tokens
	}

	// Try OKLCH path.
	if l, c, h, err := parseOKLCH(primary); err == nil {
		if _, exists := tokens["accent-hover"]; !exists {
			tokens["accent-hover"] = fmt.Sprintf("oklch(%.3f %.3f %.1f)", clamp(l-0.08, 0, 1), c, h)
		}
		if _, exists := tokens["accent-high"]; !exists {
			tokens["accent-high"] = fmt.Sprintf("oklch(%.3f %.3f %.1f)", clamp(l+0.12, 0, 1), c, h)
		}
		if _, exists := tokens["accent-low"]; !exists {
			tokens["accent-low"] = fmt.Sprintf("oklch(%.3f %.3f %.1f / 0.1)", l, c, h)
		}
		return tokens
	}

	return tokens
}

// parseHex parses a hex color string (#RGB or #RRGGBB) into RGB components.
func parseHex(hex string) (r, g, b int, err error) {
	hex = strings.TrimPrefix(hex, "#")
	switch len(hex) {
	case 3:
		rr, _ := strconv.ParseInt(string(hex[0])+string(hex[0]), 16, 32)
		gg, _ := strconv.ParseInt(string(hex[1])+string(hex[1]), 16, 32)
		bb, _ := strconv.ParseInt(string(hex[2])+string(hex[2]), 16, 32)
		return int(rr), int(gg), int(bb), nil
	case 6:
		rr, _ := strconv.ParseInt(hex[0:2], 16, 32)
		gg, _ := strconv.ParseInt(hex[2:4], 16, 32)
		bb, _ := strconv.ParseInt(hex[4:6], 16, 32)
		return int(rr), int(gg), int(bb), nil
	default:
		return 0, 0, 0, fmt.Errorf("invalid hex color: #%s", hex)
	}
}

var oklchRe = regexp.MustCompile(`^oklch\(\s*([\d.]+)(%?)\s+([\d.]+)\s+([\d.]+)\s*\)$`)

// parseOKLCH parses an oklch(L C H) string into its components.
// L can be 0-1 (decimal) or 0%-100% (percentage, converted to 0-1).
func parseOKLCH(s string) (l, c, h float64, err error) {
	m := oklchRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, 0, fmt.Errorf("invalid oklch color: %s", s)
	}
	l, err = strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, 0, 0, err
	}
	if m[2] == "%" {
		l /= 100
	}
	c, err = strconv.ParseFloat(m[3], 64)
	if err != nil {
		return 0, 0, 0, err
	}
	h, err = strconv.ParseFloat(m[4], 64)
	if err != nil {
		return 0, 0, 0, err
	}
	return l, c, h, nil
}

// rgbToHSL converts RGB (0-255) to HSL (0-360, 0-1, 0-1).
func rgbToHSL(r, g, b int) (h, s, l float64) {
	rf := float64(r) / 255
	gf := float64(g) / 255
	bf := float64(b) / 255

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	l = (max + min) / 2

	if max == min {
		return 0, 0, l
	}

	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}

	switch max {
	case rf:
		h = (gf - bf) / d
		if gf < bf {
			h += 6
		}
	case gf:
		h = (bf-rf)/d + 2
	case bf:
		h = (rf-gf)/d + 4
	}
	h *= 60

	return h, s, l
}

// hslToRGB converts HSL (0-360, 0-1, 0-1) to RGB (0-255).
func hslToRGB(h, s, l float64) (r, g, b int) {
	if s == 0 {
		v := int(math.Round(l * 255))
		return v, v, v
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	hNorm := h / 360
	r = int(math.Round(hueToRGB(p, q, hNorm+1.0/3) * 255))
	g = int(math.Round(hueToRGB(p, q, hNorm) * 255))
	b = int(math.Round(hueToRGB(p, q, hNorm-1.0/3) * 255))
	return r, g, b
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if t < 1.0/6 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2 {
		return q
	}
	if t < 2.0/3 {
		return p + (q-p)*(2.0/3-t)*6
	}
	return p
}

func toHex(r, g, b int) string {
	return fmt.Sprintf("#%02x%02x%02x", clampInt(r, 0, 255), clampInt(g, 0, 255), clampInt(b, 0, 255))
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
