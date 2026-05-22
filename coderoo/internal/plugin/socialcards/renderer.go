package socialcards

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/coderoo-dev/coderoo/internal/config"
)

const (
	cardWidth  = 1200
	cardHeight = 630
	padding    = 60
	maxContent = cardWidth - 2*padding
)

// FaceSet holds pre-created font faces for a single goroutine.
type FaceSet struct {
	Regular font.Face // 22pt
	Bold    font.Face // 56pt (may be resized per-card)
	Small   font.Face // 18pt
	Footer  font.Face // 16pt
}

// CardParams holds all data needed to render a single social card.
type CardParams struct {
	Title          string
	Description    string
	SiteTitle      string
	CollectionName string
	Date           time.Time
	BgColor        color.NRGBA
	AccentColor    color.NRGBA
	TextColor      color.NRGBA
	Faces          *FaceSet
	BoldFont       *opentype.Font
}

func newFace(f *opentype.Font, sizePt float64) (font.Face, error) {
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    sizePt,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

func newFaceSet(regularFont, boldFont *opentype.Font) (*FaceSet, error) {
	regular, err := newFace(regularFont, 22)
	if err != nil {
		return nil, err
	}
	bold, err := newFace(boldFont, 56)
	if err != nil {
		return nil, err
	}
	small, err := newFace(regularFont, 18)
	if err != nil {
		return nil, err
	}
	footer, err := newFace(regularFont, 16)
	if err != nil {
		return nil, err
	}
	return &FaceSet{Regular: regular, Bold: bold, Small: small, Footer: footer}, nil
}

func renderCard(p CardParams) *image.NRGBA {
	canvas := image.NewNRGBA(image.Rect(0, 0, cardWidth, cardHeight))

	// Background fill.
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(p.BgColor), image.Point{}, draw.Src)

	// Bottom accent strip (10px).
	stripRect := image.Rect(0, cardHeight-10, cardWidth, cardHeight)
	draw.Draw(canvas, stripRect, image.NewUniform(p.AccentColor), image.Point{}, draw.Src)

	y := padding

	// Site title (branding).
	if p.SiteTitle != "" {
		accentImg := image.NewUniform(p.AccentColor)
		d := &font.Drawer{Dst: canvas, Src: accentImg, Face: p.Faces.Small}
		d.Dot = fixed.P(padding, y+18)
		d.DrawString(p.SiteTitle)
		y += 30
	}

	// Accent bar (4px).
	barRect := image.Rect(padding, y, padding+maxContent, y+4)
	draw.Draw(canvas, barRect, image.NewUniform(p.AccentColor), image.Point{}, draw.Src)
	y += 36

	// Title (auto-sized).
	titleLines, titleFace := autoSizeTitle(p.Title, p.BoldFont, p.Faces.Bold, maxContent)
	textImg := image.NewUniform(p.TextColor)
	lineHeight := titleFace.Metrics().Height
	for _, line := range titleLines {
		y += lineHeight.Ceil()
		d := &font.Drawer{Dst: canvas, Src: textImg, Face: titleFace, Dot: fixed.P(padding, y)}
		d.DrawString(line)
		y += 4
	}

	y += 24

	// Description (up to 3 lines, 75% opacity).
	if p.Description != "" {
		descColor := color.NRGBA{R: p.TextColor.R, G: p.TextColor.G, B: p.TextColor.B, A: 191}
		descImg := image.NewUniform(descColor)
		descLines := wrapText(p.Description, p.Faces.Regular, maxContent, 3)
		descHeight := p.Faces.Regular.Metrics().Height
		for _, line := range descLines {
			y += descHeight.Ceil()
			d := &font.Drawer{Dst: canvas, Src: descImg, Face: p.Faces.Regular, Dot: fixed.P(padding, y)}
			d.DrawString(line)
		}
	}

	// Footer (collection + date, right-aligned).
	footer := buildFooter(p.CollectionName, p.Date)
	if footer != "" {
		footerColor := color.NRGBA{R: p.TextColor.R, G: p.TextColor.G, B: p.TextColor.B, A: 140}
		footerImg := image.NewUniform(footerColor)
		footerWidth := font.MeasureString(p.Faces.Footer, footer)
		footerX := cardWidth - padding - footerWidth.Ceil()
		footerY := cardHeight - 10 - 30
		d := &font.Drawer{Dst: canvas, Src: footerImg, Face: p.Faces.Footer, Dot: fixed.P(footerX, footerY)}
		d.DrawString(footer)
	}

	return canvas
}

func autoSizeTitle(title string, boldFont *opentype.Font, defaultFace font.Face, maxWidth int) ([]string, font.Face) {
	sizes := []float64{56, 44, 36}

	for _, size := range sizes {
		var face font.Face
		if size == 56 {
			face = defaultFace
		} else {
			var err error
			face, err = newFace(boldFont, size)
			if err != nil {
				face = defaultFace
			}
		}
		lines := wrapText(title, face, maxWidth, 4)
		if len(lines) <= 3 {
			return lines, face
		}
	}

	// Smallest size, cap at 3 lines with truncation.
	face, err := newFace(boldFont, 36)
	if err != nil {
		face = defaultFace
	}
	lines := wrapText(title, face, maxWidth, 3)
	return lines, face
}

func buildFooter(collectionName string, date time.Time) string {
	parts := []string{}
	if collectionName != "" {
		parts = append(parts, collectionName)
	}
	if !date.IsZero() {
		parts = append(parts, date.Format("Jan 2, 2006"))
	}
	if len(parts) == 0 {
		return ""
	}
	return joinParts(parts, " · ")
}

func joinParts(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}

func encodeImage(img *image.NRGBA, format string, quality int) ([]byte, error) {
	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
	default:
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// resolveBackground determines the card background color.
func resolveBackground(cfg map[string]any, siteCfg *config.SiteConfig) color.NRGBA {
	if override := cfgString(cfg, "bg_color", ""); override != "" {
		return parseHexColor(override)
	}
	if siteCfg != nil && siteCfg.Theme.PrimaryColor != "" {
		c := parseHexColor(siteCfg.Theme.PrimaryColor)
		return darken(c, 0.30)
	}
	return color.NRGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 0xff}
}

// resolveAccent determines the card accent color.
func resolveAccent(cfg map[string]any, siteCfg *config.SiteConfig) color.NRGBA {
	if override := cfgString(cfg, "accent_color", ""); override != "" {
		return parseHexColor(override)
	}
	if siteCfg != nil && siteCfg.Theme.AccentColor != "" {
		return parseHexColor(siteCfg.Theme.AccentColor)
	}
	return color.NRGBA{R: 0xe9, G: 0x45, B: 0x60, A: 0xff}
}

// parseHexColor parses a "#rrggbb" or "#rgb" hex string into an NRGBA color.
func parseHexColor(hex string) color.NRGBA {
	hex = trimPrefix(hex, "#")
	if len(hex) == 3 {
		r := hexVal(hex[0]) * 17
		g := hexVal(hex[1]) * 17
		b := hexVal(hex[2]) * 17
		return color.NRGBA{R: r, G: g, B: b, A: 0xff}
	}
	if len(hex) == 6 {
		r := hexVal(hex[0])*16 + hexVal(hex[1])
		g := hexVal(hex[2])*16 + hexVal(hex[3])
		b := hexVal(hex[4])*16 + hexVal(hex[5])
		return color.NRGBA{R: r, G: g, B: b, A: 0xff}
	}
	return color.NRGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 0xff}
}

func trimPrefix(s, prefix string) string {
	if len(s) > 0 && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func hexVal(b byte) uint8 {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10
	default:
		return 0
	}
}

// darken reduces the lightness of a color by the given amount (0.0-1.0).
func darken(c color.NRGBA, amount float64) color.NRGBA {
	h, s, l := rgbToHSL(c.R, c.G, c.B)
	l = math.Max(0, l-amount)
	r, g, b := hslToRGB(h, s, l)
	return color.NRGBA{R: r, G: g, B: b, A: c.A}
}

func rgbToHSL(r, g, b uint8) (h, s, l float64) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

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
	h /= 6
	return h, s, l
}

func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	if s == 0 {
		v := uint8(l * 255)
		return v, v, v
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	r := hueToRGB(p, q, h+1.0/3.0)
	g := hueToRGB(p, q, h)
	b := hueToRGB(p, q, h-1.0/3.0)

	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}
