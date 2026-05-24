package collection

import (
	"testing"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

func TestExtractVersionFromPermalink(t *testing.T) {
	vids := map[string]bool{"v1": true, "v2": true}

	tests := []struct {
		name       string
		permalink  string
		collection string
		wantVer    string
		wantRel    string
	}{
		{"versioned page", "/docs/v1/getting-started/", "docs", "v1", "getting-started"},
		{"versioned nested", "/docs/v2/guides/auth/", "docs", "v2", "guides/auth"},
		{"versioned root", "/docs/v1/", "docs", "v1", ""},
		{"unversioned page", "/docs/getting-started/", "docs", "", "getting-started"},
		{"unversioned nested", "/docs/guides/auth/", "docs", "", "guides/auth"},
		{"collection root", "/docs/", "docs", "", ""},
		{"unknown subdir", "/docs/v3/page/", "docs", "", "v3/page"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver, rel := extractVersionFromPermalink(tt.permalink, tt.collection, vids)
			if ver != tt.wantVer {
				t.Errorf("version = %q, want %q", ver, tt.wantVer)
			}
			if rel != tt.wantRel {
				t.Errorf("relPath = %q, want %q", rel, tt.wantRel)
			}
		})
	}
}

func TestRemoveVersionSegment(t *testing.T) {
	tests := []struct {
		permalink string
		version   string
		want      string
	}{
		{"/docs/v2/getting-started/", "v2", "/docs/getting-started/"},
		{"/docs/v2/", "v2", "/docs/"},
		{"/docs/v2/guides/auth/", "v2", "/docs/guides/auth/"},
	}

	for _, tt := range tests {
		got := removeVersionSegment(tt.permalink, tt.version)
		if got != tt.want {
			t.Errorf("removeVersionSegment(%q, %q) = %q, want %q", tt.permalink, tt.version, got, tt.want)
		}
	}
}

func TestAnnotateVersions(t *testing.T) {
	vc := &engine.VersionConfig{
		Enabled:     true,
		LastVersion: "v2",
		Versions: []engine.VersionDef{
			{ID: "v2", Label: "2.x", Path: "v2", Banner: "none"},
			{ID: "v1", Label: "1.x", Path: "v1", Banner: "unmaintained"},
		},
	}

	pages := []*engine.Page{
		{RelPermalink: "/docs/v2/getting-started/", Permalink: "/docs/v2/getting-started/", Kind: engine.KindPage},
		{RelPermalink: "/docs/v2/guides/auth/", Permalink: "/docs/v2/guides/auth/", Kind: engine.KindPage},
		{RelPermalink: "/docs/v1/getting-started/", Permalink: "/docs/v1/getting-started/", Kind: engine.KindPage},
		{RelPermalink: "/docs/intro/", Permalink: "/docs/intro/", Kind: engine.KindPage},
	}

	col := &engine.Collection{
		Name:   "docs",
		Config: &engine.CollectionConfig{Versioning: vc},
		Pages:  pages,
	}

	AnnotateVersions(col)

	// v2 pages should have URLs rewritten (last_version).
	if pages[0].Version != "v2" {
		t.Errorf("pages[0].Version = %q, want %q", pages[0].Version, "v2")
	}
	if pages[0].VersionRelPath != "getting-started" {
		t.Errorf("pages[0].VersionRelPath = %q, want %q", pages[0].VersionRelPath, "getting-started")
	}
	if pages[0].RelPermalink != "/docs/getting-started/" {
		t.Errorf("pages[0].RelPermalink = %q, want %q (last_version URL rewrite)", pages[0].RelPermalink, "/docs/getting-started/")
	}

	// v1 pages should keep their version prefix.
	if pages[2].Version != "v1" {
		t.Errorf("pages[2].Version = %q, want %q", pages[2].Version, "v1")
	}
	if pages[2].RelPermalink != "/docs/v1/getting-started/" {
		t.Errorf("pages[2].RelPermalink = %q, want %q", pages[2].RelPermalink, "/docs/v1/getting-started/")
	}

	// Unversioned page should have empty version.
	if pages[3].Version != "" {
		t.Errorf("pages[3].Version = %q, want empty", pages[3].Version)
	}
	if pages[3].VersionRelPath != "intro" {
		t.Errorf("pages[3].VersionRelPath = %q, want %q", pages[3].VersionRelPath, "intro")
	}
}

func TestLinkVersions(t *testing.T) {
	col := &engine.Collection{
		Name: "docs",
		Config: &engine.CollectionConfig{
			Versioning: &engine.VersionConfig{Enabled: true},
		},
	}

	p1 := &engine.Page{
		Collection:     col,
		Version:        "v1",
		VersionRelPath: "getting-started",
		Kind:           engine.KindPage,
	}
	p2 := &engine.Page{
		Collection:     col,
		Version:        "v2",
		VersionRelPath: "getting-started",
		Kind:           engine.KindPage,
	}
	p3 := &engine.Page{
		Collection:     col,
		Version:        "v1",
		VersionRelPath: "unique-page",
		Kind:           engine.KindPage,
	}

	LinkVersions([]*engine.Page{p1, p2, p3})

	if len(p1.VersionPeers) != 1 || p1.VersionPeers[0] != p2 {
		t.Errorf("p1 should have p2 as version peer, got %d peers", len(p1.VersionPeers))
	}
	if len(p2.VersionPeers) != 1 || p2.VersionPeers[0] != p1 {
		t.Errorf("p2 should have p1 as version peer, got %d peers", len(p2.VersionPeers))
	}
	if len(p3.VersionPeers) != 0 {
		t.Errorf("p3 should have no version peers, got %d", len(p3.VersionPeers))
	}
}

func TestLinkVersionsRespectsLang(t *testing.T) {
	col := &engine.Collection{
		Name: "docs",
		Config: &engine.CollectionConfig{
			Versioning: &engine.VersionConfig{Enabled: true},
		},
	}

	enV1 := &engine.Page{Collection: col, Version: "v1", VersionRelPath: "guide", Lang: "en", Kind: engine.KindPage}
	enV2 := &engine.Page{Collection: col, Version: "v2", VersionRelPath: "guide", Lang: "en", Kind: engine.KindPage}
	frV1 := &engine.Page{Collection: col, Version: "v1", VersionRelPath: "guide", Lang: "fr", Kind: engine.KindPage}
	frV2 := &engine.Page{Collection: col, Version: "v2", VersionRelPath: "guide", Lang: "fr", Kind: engine.KindPage}

	LinkVersions([]*engine.Page{enV1, enV2, frV1, frV2})

	// English v1 should link to English v2, not French v2.
	if len(enV1.VersionPeers) != 1 || enV1.VersionPeers[0] != enV2 {
		t.Errorf("enV1 peer should be enV2, got %d peers", len(enV1.VersionPeers))
	}
	if len(frV1.VersionPeers) != 1 || frV1.VersionPeers[0] != frV2 {
		t.Errorf("frV1 peer should be frV2, got %d peers", len(frV1.VersionPeers))
	}
}
