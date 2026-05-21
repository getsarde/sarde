package asset

import (
	"fmt"
	"sort"
	"strings"
)

// PictureOptions controls optional rendering behavior for <picture> elements.
type PictureOptions struct {
	IncludeDimensions bool  // when false, omit width/height from <img>
	Widths            []int // configured widths for dynamic sizes attribute
}

// RenderPicture generates HTML for a responsive <picture> element.
// If no variants are provided, falls back to a simple <img> tag.
func RenderPicture(src, alt string, width, height int, variants []ImageVariant, lqip string, lazy bool, opts ...PictureOptions) string {
	var o PictureOptions
	if len(opts) > 0 {
		o = opts[0]
	} else {
		o = PictureOptions{IncludeDimensions: true}
	}

	if len(variants) == 0 {
		return renderSimpleImg(src, alt, width, height, lazy, o.IncludeDimensions)
	}

	sizes := buildSizesAttr(o.Widths)

	var sb strings.Builder

	sb.WriteString("<picture>\n")

	// Group variants by format and write <source> elements.
	byFormat := groupByFormat(variants)
	for _, format := range sortedFormats(byFormat) {
		fmtVariants := byFormat[format]
		srcset := RenderSrcset(fmtVariants)
		mimeType := formatToMIME(format)
		sb.WriteString(fmt.Sprintf(`  <source type="%s" srcset="%s" sizes="%s">`, mimeType, srcset, sizes))
		sb.WriteByte('\n')
	}

	// <img> fallback — use the middle variant or src.
	fallbackSrc := src
	if len(variants) > 0 {
		// Pick the variant closest to 800px as the fallback.
		fallbackSrc = pickFallback(variants, 800)
	}

	sb.WriteString("  <img")
	sb.WriteString(fmt.Sprintf(` src="%s"`, fallbackSrc))
	sb.WriteString(fmt.Sprintf(` alt="%s"`, escapeAttr(alt)))
	if o.IncludeDimensions {
		if width > 0 {
			sb.WriteString(fmt.Sprintf(` width="%d"`, width))
		}
		if height > 0 {
			sb.WriteString(fmt.Sprintf(` height="%d"`, height))
		}
	}
	if lazy {
		sb.WriteString(` loading="lazy" decoding="async"`)
	}
	if lqip != "" {
		sb.WriteString(fmt.Sprintf(` style="background-image: url(%s); background-size: cover;"`, lqip))
	}
	sb.WriteString(">\n")

	sb.WriteString("</picture>")
	return sb.String()
}

// RenderSrcset generates the srcset attribute value from a list of variants.
func RenderSrcset(variants []ImageVariant) string {
	// Sort by width ascending.
	sorted := make([]ImageVariant, len(variants))
	copy(sorted, variants)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Width < sorted[j].Width })

	var parts []string
	for _, v := range sorted {
		parts = append(parts, fmt.Sprintf("%s %dw", v.URL, v.Width))
	}
	return strings.Join(parts, ", ")
}

func renderSimpleImg(src, alt string, width, height int, lazy, includeDimensions bool) string {
	var sb strings.Builder
	sb.WriteString("<img")
	sb.WriteString(fmt.Sprintf(` src="%s"`, src))
	sb.WriteString(fmt.Sprintf(` alt="%s"`, escapeAttr(alt)))
	if includeDimensions {
		if width > 0 {
			sb.WriteString(fmt.Sprintf(` width="%d"`, width))
		}
		if height > 0 {
			sb.WriteString(fmt.Sprintf(` height="%d"`, height))
		}
	}
	if lazy {
		sb.WriteString(` loading="lazy" decoding="async"`)
	}
	sb.WriteString(">")
	return sb.String()
}

// buildSizesAttr generates a sizes attribute from configured widths.
// Falls back to a sensible default when no widths are provided.
func buildSizesAttr(widths []int) string {
	if len(widths) == 0 {
		return "(max-width: 600px) 400px, (max-width: 1024px) 800px, 1200px"
	}

	sorted := make([]int, len(widths))
	copy(sorted, widths)
	sort.Ints(sorted)

	if len(sorted) == 1 {
		return fmt.Sprintf("%dpx", sorted[0])
	}

	var parts []string
	for i := 0; i < len(sorted)-1; i++ {
		breakpoint := sorted[i] + sorted[i]*50/100
		parts = append(parts, fmt.Sprintf("(max-width: %dpx) %dpx", breakpoint, sorted[i]))
	}
	parts = append(parts, fmt.Sprintf("%dpx", sorted[len(sorted)-1]))
	return strings.Join(parts, ", ")
}

func groupByFormat(variants []ImageVariant) map[string][]ImageVariant {
	m := make(map[string][]ImageVariant)
	for _, v := range variants {
		m[v.Format] = append(m[v.Format], v)
	}
	return m
}

func sortedFormats(byFormat map[string][]ImageVariant) []string {
	// Put webp/avif first (preferred), then original formats.
	var formats []string
	for f := range byFormat {
		formats = append(formats, f)
	}
	sort.Slice(formats, func(i, j int) bool {
		pi := formatPriority(formats[i])
		pj := formatPriority(formats[j])
		return pi < pj
	})
	return formats
}

func formatPriority(f string) int {
	switch f {
	case "avif":
		return 0
	case "webp":
		return 1
	default:
		return 2
	}
}

func formatToMIME(format string) string {
	switch format {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "avif":
		return "image/avif"
	case "gif":
		return "image/gif"
	default:
		return "image/" + format
	}
}

func pickFallback(variants []ImageVariant, targetWidth int) string {
	best := variants[0]
	bestDist := abs(best.Width - targetWidth)
	for _, v := range variants[1:] {
		d := abs(v.Width - targetWidth)
		if d < bestDist {
			best = v
			bestDist = d
		}
	}
	return best.URL
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
