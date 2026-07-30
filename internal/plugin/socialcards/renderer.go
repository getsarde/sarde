package socialcards

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

const (
	cardWidth  = 1200
	cardHeight = 630
	padding    = 64 // 8px grid
	maxContent = cardWidth - 2*padding

	logoDrawSize   = 64 // on-card logo box, px
	logoTextGap    = 16 // gap between logo and site title
	brandingGapMin = 24 // min gap between branding row and bottom block

	titleLineGap = 4  // extra leading between title lines
	gapTitleDesc = 20 // gap between title block and description
	gapDescFooter = 24 // gap between description and footer

	accentStripW = 180 // bottom-right partial accent strip width
	accentStripH = 3

	watermarkLongEdge       = 820  // watermark image long edge, px
	watermarkBleed          = 200  // how far the watermark hangs off the right edge
	watermarkOpacityDefault = 0.07

	gradientAmount = 0.08 // background gradient lightness delta
)

// FaceSet holds pre-created font faces for a single goroutine. Sizes follow
// OG-image readability guidance: cards render at roughly 40% scale in feeds
// and 200px wide as thumbnails, so nothing important goes below ~20px.
type FaceSet struct {
	Small  font.Face // 20pt site title, next to a logo mark
	Brand  font.Face // 28pt bold site title, standing alone (no logo)
	Desc   font.Face // 32pt description
	Bold   font.Face // 76pt title, first ladder rung (may be resized per-card)
	Footer font.Face // 20pt footer
}

// CardParams holds all data needed to render a single social card.
type CardParams struct {
	Title          string
	Description    string
	SiteTitle      string
	CollectionName string
	Date           time.Time
	// DateExplicit gates the footer date: an inferred (mtime) date is build
	// metadata, not a publish date, and is never shown.
	DateExplicit bool
	BgColor      color.NRGBA
	AccentColor  color.NRGBA
	// AccentColor2 turns the accent strip into a two-stop horizontal
	// gradient when non-nil.
	AccentColor2 *color.NRGBA
	TextColor    color.NRGBA
	// LogoImage is the brand mark drawn top-left, already resized to
	// logoDrawSize. Nil draws no logo.
	LogoImage *image.NRGBA
	// WatermarkImage is the large low-opacity mark bleeding off the right
	// edge, already resized to watermarkLongEdge. Nil draws no watermark.
	WatermarkImage   *image.NRGBA
	WatermarkOpacity float64
	Faces            *FaceSet
	BoldFont         *opentype.Font
}

func newFace(f *opentype.Font, sizePt float64) (font.Face, error) {
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    sizePt,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

func newFaceSet(regularFont, boldFont *opentype.Font) (*FaceSet, error) {
	desc, err := newFace(regularFont, 32)
	if err != nil {
		return nil, err
	}
	bold, err := newFace(boldFont, 76)
	if err != nil {
		return nil, err
	}
	small, err := newFace(regularFont, 20)
	if err != nil {
		return nil, err
	}
	brand, err := newFace(boldFont, 28)
	if err != nil {
		return nil, err
	}
	footer, err := newFace(regularFont, 20)
	if err != nil {
		return nil, err
	}
	return &FaceSet{Small: small, Brand: brand, Desc: desc, Bold: bold, Footer: footer}, nil
}

func renderCard(p CardParams) *image.NRGBA {
	canvas := image.NewNRGBA(image.Rect(0, 0, cardWidth, cardHeight))

	// Background: subtle vertical gradient. Dark backgrounds deepen toward
	// the bottom so the bottom-anchored light text gets more contrast (and
	// the card reads as dark as its configured color); light backgrounds
	// brighten instead, for the same contrast reason with dark text. Row 0
	// always equals the configured background exactly, so a logo whose tile
	// matches bg_color sits seamlessly in the top corner.
	for row := 0; row < cardHeight; row++ {
		rowColor := gradientRowColor(p.BgColor, row, cardHeight, gradientAmount)
		draw.Draw(canvas, image.Rect(0, row, cardWidth, row+1), image.NewUniform(rowColor), image.Point{}, draw.Src)
	}

	// Watermark: large low-opacity mark bleeding off the right edge. Drawn
	// before all text so content always sits on top.
	if p.WatermarkImage != nil {
		opacity := p.WatermarkOpacity
		if opacity <= 0 {
			opacity = watermarkOpacityDefault
		}
		wb := p.WatermarkImage.Bounds()
		pos := image.Pt(cardWidth-wb.Dx()+watermarkBleed, (cardHeight-wb.Dy())/2)
		canvas = imaging.Overlay(canvas, p.WatermarkImage, pos, opacity)
	}

	// Branding row, top-left: optional logo mark plus site title.
	brandingBottom := padding
	if p.LogoImage != nil {
		lb := p.LogoImage.Bounds()
		canvas = imaging.Overlay(canvas, p.LogoImage, image.Pt(padding, padding), 1.0)
		if bottom := padding + lb.Dy(); bottom > brandingBottom {
			brandingBottom = bottom
		}
	}
	if p.SiteTitle != "" {
		// Next to a logo the site title is a small annotation; standing
		// alone it is the whole brand presence, so it renders as a larger
		// bold wordmark instead.
		face := p.Faces.Small
		if p.LogoImage == nil {
			face = p.Faces.Brand
		}
		metrics := face.Metrics()
		textX := padding
		var baseline int
		if p.LogoImage != nil {
			// Vertically center the text against the logo box.
			textX = padding + p.LogoImage.Bounds().Dx() + logoTextGap
			baseline = padding + (p.LogoImage.Bounds().Dy()+metrics.Ascent.Ceil()-metrics.Descent.Ceil())/2
		} else {
			baseline = padding + metrics.Ascent.Ceil()
			if bottom := padding + metrics.Height.Ceil(); bottom > brandingBottom {
				brandingBottom = bottom
			}
		}
		d := &font.Drawer{Dst: canvas, Src: image.NewUniform(p.AccentColor), Face: face}
		d.Dot = fixed.P(textX, baseline)
		d.DrawString(p.SiteTitle)
	}

	// Bottom-anchored content block: measure first, then draw downward from
	// the computed start.
	titleLines, titleFace := autoSizeTitle(p.Title, p.BoldFont, p.Faces.Bold, maxContent)
	titleLineH := titleFace.Metrics().Height.Ceil() + titleLineGap
	blockH := len(titleLines) * titleLineH

	var descLines []string
	descLineH := p.Faces.Desc.Metrics().Height.Ceil()
	if p.Description != "" {
		descLines = wrapText(p.Description, p.Faces.Desc, maxContent, 2)
		if len(descLines) > 0 {
			blockH += gapTitleDesc + len(descLines)*descLineH
		}
	}

	footer := buildFooter(p.CollectionName, p.Date, p.DateExplicit)
	footerLineH := p.Faces.Footer.Metrics().Height.Ceil()
	if footer != "" {
		blockH += gapDescFooter + footerLineH
	}

	y := (cardHeight - padding) - blockH
	// Never overlap the branding row: extreme content degrades to starting
	// right below it instead of true bottom anchoring.
	if minY := brandingBottom + brandingGapMin; y < minY {
		y = minY
	}

	textImg := image.NewUniform(p.TextColor)
	for _, line := range titleLines {
		y += titleFace.Metrics().Height.Ceil()
		d := &font.Drawer{Dst: canvas, Src: textImg, Face: titleFace, Dot: fixed.P(padding, y)}
		d.DrawString(line)
		y += titleLineGap
	}

	if len(descLines) > 0 {
		y += gapTitleDesc
		descColor := color.NRGBA{R: p.TextColor.R, G: p.TextColor.G, B: p.TextColor.B, A: 200}
		descImg := image.NewUniform(descColor)
		for _, line := range descLines {
			y += descLineH
			d := &font.Drawer{Dst: canvas, Src: descImg, Face: p.Faces.Desc, Dot: fixed.P(padding, y)}
			d.DrawString(line)
		}
	}

	if footer != "" {
		y += gapDescFooter + footerLineH
		footerColor := color.NRGBA{R: p.TextColor.R, G: p.TextColor.G, B: p.TextColor.B, A: 130}
		d := &font.Drawer{Dst: canvas, Src: image.NewUniform(footerColor), Face: p.Faces.Footer, Dot: fixed.P(padding, y)}
		d.DrawString(footer)
	}

	// Partial-width accent strip, flush to the bottom-right padding corner.
	stripRect := image.Rect(cardWidth-padding-accentStripW, cardHeight-accentStripH, cardWidth-padding, cardHeight)
	drawAccentStrip(canvas, stripRect, p.AccentColor, p.AccentColor2)

	return canvas
}

// gradientRowColor returns the background color for one row of the vertical
// gradient. Row 0 is exactly bg; the last row is bg with its HSL lightness
// shifted by amount, away from the text color's end of the scale: dark
// backgrounds darken toward the bottom and light backgrounds lighten, so the
// bottom-anchored text always gains contrast rather than losing it.
func gradientRowColor(bg color.NRGBA, row, totalRows int, amount float64) color.NRGBA {
	// Short-circuit the base row: the RGB->HSL->RGB round trip is lossy, and
	// the top edge must match the configured background exactly.
	if totalRows <= 1 || row == 0 {
		return bg
	}
	h, s, l := rgbToHSL(bg.R, bg.G, bg.B)
	endL := l - amount
	if l > 0.5 {
		endL = l + amount
	}
	endL = math.Max(0, math.Min(1, endL))
	t := float64(row) / float64(totalRows-1)
	r, g, b := hslToRGB(h, s, l+(endL-l)*t)
	return color.NRGBA{R: r, G: g, B: b, A: bg.A}
}

// drawAccentStrip fills rect with c1, or with a horizontal left-to-right
// linear blend from c1 to c2 when c2 is non-nil.
func drawAccentStrip(canvas *image.NRGBA, rect image.Rectangle, c1 color.NRGBA, c2 *color.NRGBA) {
	if c2 == nil {
		draw.Draw(canvas, rect, image.NewUniform(c1), image.Point{}, draw.Src)
		return
	}
	w := rect.Dx()
	if w <= 1 {
		draw.Draw(canvas, rect, image.NewUniform(c1), image.Point{}, draw.Src)
		return
	}
	for x := 0; x < w; x++ {
		t := float64(x) / float64(w-1)
		col := lerpColor(c1, *c2, t)
		draw.Draw(canvas, image.Rect(rect.Min.X+x, rect.Min.Y, rect.Min.X+x+1, rect.Max.Y), image.NewUniform(col), image.Point{}, draw.Src)
	}
}

// lerpColor linearly interpolates between two colors in RGB space, matching
// how SVG linear gradients interpolate their stops.
func lerpColor(a, b color.NRGBA, t float64) color.NRGBA {
	lerp := func(x, y uint8) uint8 {
		return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t))
	}
	return color.NRGBA{R: lerp(a.R, b.R), G: lerp(a.G, b.G), B: lerp(a.B, b.B), A: lerp(a.A, b.A)}
}

func autoSizeTitle(title string, boldFont *opentype.Font, defaultFace font.Face, maxWidth int) ([]string, font.Face) {
	sizes := []float64{76, 60, 48}

	for _, size := range sizes {
		var face font.Face
		if size == sizes[0] {
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
	face, err := newFace(boldFont, sizes[len(sizes)-1])
	if err != nil {
		face = defaultFace
	}
	lines := wrapText(title, face, maxWidth, 3)
	return lines, face
}

// buildFooter joins the collection name and, only when it was explicitly
// authored (frontmatter or filename prefix), the page date. Inferred mtime
// dates are never shown: they are build metadata, not publish dates.
func buildFooter(collectionName string, date time.Time, dateExplicit bool) string {
	parts := []string{}
	if collectionName != "" {
		parts = append(parts, collectionName)
	}
	if dateExplicit && !date.IsZero() {
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
	if override := cfgutil.String(cfg,"bg_color", ""); override != "" {
		return parseHexColor(override)
	}
	if siteCfg != nil {
		accentVal := siteCfg.Theme.AccentColor
		if accentVal == "" {
			accentVal = siteCfg.Theme.PrimaryColor
		}
		if accentVal != "" {
			c := parseHexColor(accentVal)
			return darken(c, 0.30)
		}
	}
	return color.NRGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 0xff}
}

// resolveAccent determines the card accent color.
func resolveAccent(cfg map[string]any, siteCfg *config.SiteConfig) color.NRGBA {
	if override := cfgutil.String(cfg,"accent_color", ""); override != "" {
		return parseHexColor(override)
	}
	if siteCfg != nil && siteCfg.Theme.AccentColor != "" {
		return parseHexColor(siteCfg.Theme.AccentColor)
	}
	return color.NRGBA{R: 0xe9, G: 0x45, B: 0x60, A: 0xff}
}

// parseHexColor parses a "#rrggbb" or "#rgb" hex string into an NRGBA color.
func parseHexColor(hex string) color.NRGBA {
	hex = strings.TrimPrefix(hex, "#")
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

// lighten increases the lightness of a color by the given amount (0.0-1.0).
func lighten(c color.NRGBA, amount float64) color.NRGBA {
	h, s, l := rgbToHSL(c.R, c.G, c.B)
	l = math.Min(1, l+amount)
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
