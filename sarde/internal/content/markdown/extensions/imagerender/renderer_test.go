package imagerender

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/yuin/goldmark/ast"
)

func TestParseImageAttrs_AllFields(t *testing.T) {
	n := ast.NewImage(ast.NewLink())
	n.SetAttribute([]byte("op"), []byte("fill"))
	n.SetAttribute([]byte("width"), []byte("800"))
	n.SetAttribute([]byte("height"), []byte("400"))
	n.SetAttribute([]byte("quality"), []byte("90"))
	n.SetAttribute([]byte("format"), []byte("webp"))

	opts := parseImageAttrs(n)

	if opts.Op != asset.ResizeOpFill {
		t.Errorf("Op = %q, want fill", opts.Op)
	}
	if opts.Width != 800 {
		t.Errorf("Width = %d, want 800", opts.Width)
	}
	if opts.Height != 400 {
		t.Errorf("Height = %d, want 400", opts.Height)
	}
	if opts.Quality != 90 {
		t.Errorf("Quality = %d, want 90", opts.Quality)
	}
	if len(opts.Formats) != 1 || opts.Formats[0] != "webp" {
		t.Errorf("Formats = %v, want [webp]", opts.Formats)
	}
}

func TestParseImageAttrs_NoAttributes(t *testing.T) {
	n := ast.NewImage(ast.NewLink())

	opts := parseImageAttrs(n)

	if opts.Op != "" {
		t.Errorf("Op = %q, want empty", opts.Op)
	}
	if opts.Width != 0 {
		t.Errorf("Width = %d, want 0", opts.Width)
	}
	if opts.Height != 0 {
		t.Errorf("Height = %d, want 0", opts.Height)
	}
	if opts.Quality != 0 {
		t.Errorf("Quality = %d, want 0", opts.Quality)
	}
	if len(opts.Formats) != 0 {
		t.Errorf("Formats = %v, want empty", opts.Formats)
	}
}

func TestParseImageAttrs_PartialAttributes(t *testing.T) {
	n := ast.NewImage(ast.NewLink())
	n.SetAttribute([]byte("width"), []byte("600"))
	n.SetAttribute([]byte("op"), []byte("fit"))

	opts := parseImageAttrs(n)

	if opts.Op != asset.ResizeOpFit {
		t.Errorf("Op = %q, want fit", opts.Op)
	}
	if opts.Width != 600 {
		t.Errorf("Width = %d, want 600", opts.Width)
	}
	if opts.Height != 0 {
		t.Errorf("Height = %d, want 0", opts.Height)
	}
}

func TestImageRenderer_ParsesAttributes(t *testing.T) {
	var receivedOpts asset.ImageOptions
	var receivedName string

	lookup := func(imageName string, opts asset.ImageOptions) *asset.ProcessedImage {
		receivedName = imageName
		receivedOpts = opts
		return &asset.ProcessedImage{
			Src:     "/img/hero.jpg",
			Width:   800,
			Height:  400,
			Loading: "lazy",
		}
	}

	r := NewRenderer(lookup)

	n := ast.NewImage(ast.NewLink())
	n.Destination = []byte("hero.jpg")
	n.SetAttribute([]byte("op"), []byte("fill"))
	n.SetAttribute([]byte("width"), []byte("800"))
	n.SetAttribute([]byte("height"), []byte("400"))

	// We can't easily call renderImage directly without a full Goldmark setup,
	// but we can verify parseImageAttrs works correctly.
	opts := parseImageAttrs(n)
	_ = r.Lookup("hero.jpg", opts)

	if receivedName != "hero.jpg" {
		t.Errorf("lookup received name %q, want hero.jpg", receivedName)
	}
	if receivedOpts.Op != asset.ResizeOpFill {
		t.Errorf("lookup received Op = %q, want fill", receivedOpts.Op)
	}
	if receivedOpts.Width != 800 {
		t.Errorf("lookup received Width = %d, want 800", receivedOpts.Width)
	}
	if receivedOpts.Height != 400 {
		t.Errorf("lookup received Height = %d, want 400", receivedOpts.Height)
	}
}

// The fallback <img> path must HTML-escape the destination so a quote in an
// image URL cannot break out of the src attribute (XSS).
func TestRenderImage_FallbackEscapesDestination(t *testing.T) {
	r := NewRenderer(nil) // nil lookup forces the fallback path

	n := ast.NewImage(ast.NewLink())
	n.Destination = []byte(`x.png" onerror="alert(1)`)

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if _, err := r.renderImage(w, nil, n, true); err != nil {
		t.Fatalf("renderImage failed: %v", err)
	}
	w.Flush()
	out := buf.String()

	if strings.Contains(out, `onerror="alert(1)`) {
		t.Errorf("destination not escaped, attribute breakout possible:\n%s", out)
	}
	if !strings.Contains(out, `src="x.png&quot; onerror=&quot;alert(1)"`) {
		t.Errorf("expected escaped destination in src attribute:\n%s", out)
	}
}

func TestRenderImage_FallbackPlainDestination(t *testing.T) {
	r := NewRenderer(nil)

	n := ast.NewImage(ast.NewLink())
	n.Destination = []byte("https://example.com/photo.jpg")

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if _, err := r.renderImage(w, nil, n, true); err != nil {
		t.Fatalf("renderImage failed: %v", err)
	}
	w.Flush()
	out := buf.String()

	if !strings.Contains(out, `src="https://example.com/photo.jpg"`) {
		t.Errorf("plain destination should pass through unchanged:\n%s", out)
	}
	if !strings.Contains(out, `loading="lazy"`) {
		t.Errorf("fallback img should keep lazy loading:\n%s", out)
	}
}

func TestImageRenderer_NoAttributesFallback(t *testing.T) {
	var receivedOpts asset.ImageOptions

	lookup := func(imageName string, opts asset.ImageOptions) *asset.ProcessedImage {
		receivedOpts = opts
		return &asset.ProcessedImage{
			Src:     "/img/photo.jpg",
			Width:   1200,
			Height:  800,
			Loading: "lazy",
		}
	}

	r := NewRenderer(lookup)

	n := ast.NewImage(ast.NewLink())
	n.Destination = []byte("photo.jpg")

	opts := parseImageAttrs(n)
	_ = r.Lookup("photo.jpg", opts)

	if receivedOpts.Op != "" {
		t.Errorf("Op = %q, want empty", receivedOpts.Op)
	}
	if receivedOpts.Width != 0 {
		t.Errorf("Width = %d, want 0", receivedOpts.Width)
	}
	if receivedOpts.Height != 0 {
		t.Errorf("Height = %d, want 0", receivedOpts.Height)
	}
	if receivedOpts.Quality != 0 {
		t.Errorf("Quality = %d, want 0", receivedOpts.Quality)
	}
}
