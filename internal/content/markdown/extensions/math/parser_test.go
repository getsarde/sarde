package math

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func renderMathMarkdown(t *testing.T, md string) string {
	t.Helper()
	gm := goldmark.New(goldmark.WithExtensions(&Extension{}))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

// Prose with two currency amounts must not be parsed as an inline math span.
func TestInlineMath_TwoDollarAmountsNotMath(t *testing.T) {
	out := renderMathMarkdown(t, "The plan costs $9 and the pro plan costs $29 per month.\n")
	if strings.Contains(out, "sarde-math") {
		t.Errorf("dollar amounts must not render as math:\n%s", out)
	}
}

// A genuine inline math span still renders.
func TestInlineMath_RealExpressionStillMatches(t *testing.T) {
	out := renderMathMarkdown(t, "The value $x + y$ is computed.\n")
	if !strings.Contains(out, "sarde-math-inline") {
		t.Errorf("expected inline math span:\n%s", out)
	}
}
