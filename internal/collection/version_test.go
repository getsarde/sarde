package collection

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func TestLangVersionKey(t *testing.T) {
	tests := []struct {
		lang, ver, want string
	}{
		{"en", "v1", "en/v1"},
		{"fr", "v2", "fr/v2"},
		{"en", "", "en/_latest"},
		{"", "v1", "_default/v1"},
		{"", "", "_default/_latest"},
	}
	for _, tt := range tests {
		got := LangVersionKey(tt.lang, tt.ver)
		if got != tt.want {
			t.Errorf("LangVersionKey(%q, %q) = %q, want %q", tt.lang, tt.ver, got, tt.want)
		}
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
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Collection: col},
		PageVersioning:    engine.PageVersioning{Version: "v1", VersionRelPath: "getting-started"},
	}
	p2 := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Collection: col},
		PageVersioning:    engine.PageVersioning{Version: "v2", VersionRelPath: "getting-started"},
	}
	p3 := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Collection: col},
		PageVersioning:    engine.PageVersioning{Version: "v1", VersionRelPath: "unique-page"},
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

	enV1 := &engine.Page{PageIdentity: engine.PageIdentity{Kind: engine.KindPage}, PageRelationships: engine.PageRelationships{Collection: col}, PageVersioning: engine.PageVersioning{Version: "v1", VersionRelPath: "guide"}, PageI18n: engine.PageI18n{Lang: "en"}}
	enV2 := &engine.Page{PageIdentity: engine.PageIdentity{Kind: engine.KindPage}, PageRelationships: engine.PageRelationships{Collection: col}, PageVersioning: engine.PageVersioning{Version: "v2", VersionRelPath: "guide"}, PageI18n: engine.PageI18n{Lang: "en"}}
	frV1 := &engine.Page{PageIdentity: engine.PageIdentity{Kind: engine.KindPage}, PageRelationships: engine.PageRelationships{Collection: col}, PageVersioning: engine.PageVersioning{Version: "v1", VersionRelPath: "guide"}, PageI18n: engine.PageI18n{Lang: "fr"}}
	frV2 := &engine.Page{PageIdentity: engine.PageIdentity{Kind: engine.KindPage}, PageRelationships: engine.PageRelationships{Collection: col}, PageVersioning: engine.PageVersioning{Version: "v2", VersionRelPath: "guide"}, PageI18n: engine.PageI18n{Lang: "fr"}}

	LinkVersions([]*engine.Page{enV1, enV2, frV1, frV2})

	if len(enV1.VersionPeers) != 1 || enV1.VersionPeers[0] != enV2 {
		t.Errorf("enV1 peer should be enV2, got %d peers", len(enV1.VersionPeers))
	}
	if len(frV1.VersionPeers) != 1 || frV1.VersionPeers[0] != frV2 {
		t.Errorf("frV1 peer should be frV2, got %d peers", len(frV1.VersionPeers))
	}
}
