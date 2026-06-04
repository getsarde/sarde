package icons

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	reSVGOpen  = regexp.MustCompile(`(?is)<svg\b([^>]*)>`)
	reSVGClose = regexp.MustCompile(`(?is)</svg\s*>`)
	reAttrVB   = regexp.MustCompile(`(?is)\bviewBox\s*=\s*["']([^"']*)["']`)
	reAttrW    = regexp.MustCompile(`(?is)\bwidth\s*=\s*["']([^"']*)["']`)
	reAttrH    = regexp.MustCompile(`(?is)\bheight\s*=\s*["']([^"']*)["']`)
	reScript   = regexp.MustCompile(`(?is)<script\b.*?</script\s*>`)
	reOnAttr   = regexp.MustCompile(`(?is)\s+on[a-z]+\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
)

// parseSVGFile strips the outer <svg> wrapper from an SVG file, returning the
// inner body, the viewBox string, and the width/height (numeric, 0 when absent
// or non-numeric). If the file has no viewBox, one is synthesized from
// width/height. Local project SVGs are author-trusted, so sanitization is
// deliberately light — it drops <script> elements and on* event-handler
// attributes, not a full CSP-aware sanitizer. Returns an error if no <svg> root
// is found or the body is empty.
//
// Attribute extraction uses `[^>]` matching, so a root-tag attribute value that
// itself contains `>` is not supported (vanishingly rare for icon files).
func parseSVGFile(data []byte) (body, viewBox string, width, height int, err error) {
	s := string(data)
	open := reSVGOpen.FindStringSubmatchIndex(s)
	if open == nil {
		return "", "", 0, 0, fmt.Errorf("no <svg> element found")
	}
	attrs := s[open[2]:open[3]]
	inner := s[open[1]:]
	// Trim at the LAST </svg> so a (rare) nested <svg> doesn't close early.
	if loc := reSVGClose.FindAllStringIndex(inner, -1); loc != nil {
		inner = inner[:loc[len(loc)-1][0]]
	}

	if m := reAttrVB.FindStringSubmatch(attrs); m != nil {
		viewBox = strings.TrimSpace(m[1])
	}
	width = parseDimAttr(reAttrW.FindStringSubmatch(attrs))
	height = parseDimAttr(reAttrH.FindStringSubmatch(attrs))

	inner = reScript.ReplaceAllString(inner, "")
	inner = reOnAttr.ReplaceAllString(inner, "")

	body = strings.TrimSpace(inner)
	if body == "" {
		return "", "", 0, 0, fmt.Errorf("empty <svg> body")
	}

	if viewBox == "" && width > 0 && height > 0 {
		viewBox = fmt.Sprintf("0 0 %d %d", width, height)
	}
	return body, viewBox, width, height, nil
}

// parseDimAttr reads the leading numeric part of a width/height attr match
// (e.g. "24px" → 24). Returns 0 when absent or unparseable.
func parseDimAttr(m []string) int {
	if m == nil {
		return 0
	}
	num, _ := splitNumUnit(strings.TrimSpace(m[1]))
	if num == "" {
		return 0
	}
	f, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0
	}
	return int(f)
}
