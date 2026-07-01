package cardgrid_test

import (
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/content/markdown"
)

func TestCardGridStaggerBareFlag(t *testing.T) {
	r := markdown.NewRenderer()
	md := ":::card-grid{stagger}\n:::card[A]\nBody.\n:::\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(result.HTML, "sarde-card-grid-2") {
		t.Errorf("expected sarde-card-grid-2 class, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "sarde-card-grid-stagger") {
		t.Errorf("expected sarde-card-grid-stagger class, got: %s", result.HTML)
	}
}

func TestCardGridStaggerOverridesCols(t *testing.T) {
	r := markdown.NewRenderer()
	md := ":::card-grid{cols=\"3\" stagger}\n:::card[A]\nBody.\n:::\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(result.HTML, "sarde-card-grid-2") {
		t.Errorf("expected stagger to force sarde-card-grid-2, got: %s", result.HTML)
	}
	if strings.Contains(result.HTML, "sarde-card-grid-3") {
		t.Errorf("expected cols=3 to be overridden by stagger, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "sarde-card-grid-stagger") {
		t.Errorf("expected sarde-card-grid-stagger class, got: %s", result.HTML)
	}
}

func TestCardGridColsWithoutStagger(t *testing.T) {
	r := markdown.NewRenderer()
	md := ":::card-grid{cols=\"3\"}\n:::card[A]\nBody.\n:::\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(result.HTML, "sarde-card-grid-3") {
		t.Errorf("expected sarde-card-grid-3 class, got: %s", result.HTML)
	}
	if strings.Contains(result.HTML, "sarde-card-grid-stagger") {
		t.Errorf("did not expect sarde-card-grid-stagger class, got: %s", result.HTML)
	}
}

func TestCardGridNoAttrs(t *testing.T) {
	r := markdown.NewRenderer()
	md := ":::card-grid\n:::card[A]\nBody.\n:::\n:::"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	for _, cls := range []string{"sarde-card-grid-2", "sarde-card-grid-3", "sarde-card-grid-4", "sarde-card-grid-stagger"} {
		if strings.Contains(result.HTML, cls) {
			t.Errorf("did not expect %s class, got: %s", cls, result.HTML)
		}
	}
}
