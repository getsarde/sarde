package collection

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func TestInferCollection_Blog(t *testing.T) {
	for _, name := range []string{"blog", "posts", "articles", "news"} {
		cfg := InferCollection(name)
		if cfg.SortBy != "date" {
			t.Errorf("%s: SortBy = %q, want %q", name, cfg.SortBy, "date")
		}
		if cfg.SortOrder != "desc" {
			t.Errorf("%s: SortOrder = %q, want %q", name, cfg.SortOrder, "desc")
		}
		if cfg.Layout != engine.LayoutDefault {
			t.Errorf("%s: Layout = %q, want %q", name, cfg.Layout, engine.LayoutDefault)
		}
		if !cfg.Feed {
			t.Errorf("%s: Feed should be true", name)
		}
		if cfg.Paginate != 10 {
			t.Errorf("%s: Paginate = %d, want 10", name, cfg.Paginate)
		}
	}
}

func TestInferCollection_Docs(t *testing.T) {
	for _, name := range []string{"docs", "documentation", "courses", "tutorials", "guides", "reference", "lessons", "workshops"} {
		cfg := InferCollection(name)
		if cfg.SortBy != "order" {
			t.Errorf("%s: SortBy = %q, want %q", name, cfg.SortBy, "order")
		}
		if cfg.SortOrder != "asc" {
			t.Errorf("%s: SortOrder = %q, want %q", name, cfg.SortOrder, "asc")
		}
		if cfg.Layout != engine.LayoutDocs {
			t.Errorf("%s: Layout = %q, want %q", name, cfg.Layout, engine.LayoutDocs)
		}
		if cfg.Sidebar == nil {
			t.Errorf("%s: Sidebar should not be nil", name)
		}
		if cfg.TOC == nil {
			t.Errorf("%s: TOC should not be nil", name)
		}
		if cfg.PrevNext == nil || !cfg.PrevNext.Enabled {
			t.Errorf("%s: PrevNext should be enabled", name)
		}
	}
}

func TestInferCollection_Unknown(t *testing.T) {
	cfg := InferCollection("projects")
	if cfg.SortBy != "title" {
		t.Errorf("SortBy = %q, want %q", cfg.SortBy, "title")
	}
	if cfg.SortOrder != "asc" {
		t.Errorf("SortOrder = %q, want %q", cfg.SortOrder, "asc")
	}
	if cfg.Layout != engine.LayoutDefault {
		t.Errorf("Layout = %q, want %q", cfg.Layout, engine.LayoutDefault)
	}
	if cfg.Feed {
		t.Error("Feed should be false")
	}
	if cfg.Paginate != 0 {
		t.Errorf("Paginate = %d, want 0", cfg.Paginate)
	}
}
