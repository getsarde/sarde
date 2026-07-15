package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func writeSidebarYAML(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "sidebar.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSidebarFile_Absent(t *testing.T) {
	sf, err := LoadSidebarFile(t.TempDir())
	if err != nil {
		t.Fatalf("absent file must not error: %v", err)
	}
	if sf != nil {
		t.Fatalf("absent file must return nil, got %v", sf)
	}
}

func TestLoadSidebarFile_Valid(t *testing.T) {
	dir := t.TempDir()
	writeSidebarYAML(t, dir, `
docs:
  collapse_level: 2
  tabs:
    guide:
      label: "Getting Started"
      icon: book-open
      order: 10
      description: "Learn the basics."
  overrides:
    guide/advanced:
      label: "Advanced Topics"
      collapsed: true
      order: 20
    guide/introduction:
      badge: "New"
    plugins/internal:
      hidden: true
    api/router:
      badge:
        text: "WIP"
        variant: caution
      attrs:
        data-x: "y"
`)

	sf, err := LoadSidebarFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	entry := sf["docs"]
	if entry == nil {
		t.Fatal("expected docs entry")
	}

	if entry.CollapseLevel == nil || *entry.CollapseLevel != 2 {
		t.Errorf("collapse_level: got %v", entry.CollapseLevel)
	}

	tab := entry.Tabs["guide"]
	if tab == nil {
		t.Fatal("expected guide tab override")
	}
	if tab.Label != "Getting Started" || tab.Icon != "book-open" || tab.Description != "Learn the basics." {
		t.Errorf("tab fields: got %+v", tab)
	}
	if tab.Order == nil || *tab.Order != 10 {
		t.Errorf("tab order: got %v", tab.Order)
	}

	adv := entry.Overrides["guide/advanced"]
	if adv == nil {
		t.Fatal("expected guide/advanced override")
	}
	if adv.Label != "Advanced Topics" {
		t.Errorf("label: got %q", adv.Label)
	}
	if adv.Collapsed == nil || !*adv.Collapsed {
		t.Errorf("collapsed: got %v", adv.Collapsed)
	}
	if adv.Order == nil || *adv.Order != 20 {
		t.Errorf("order: got %v", adv.Order)
	}

	// Scalar badge form.
	intro := entry.Overrides["guide/introduction"]
	if intro == nil || intro.Badge.Text != "New" || intro.Badge.Variant != engine.BadgeVariantDefault {
		t.Errorf("scalar badge: got %+v", intro)
	}

	// Object badge form plus attrs.
	router := entry.Overrides["api/router"]
	if router == nil || router.Badge.Text != "WIP" || router.Badge.Variant != engine.BadgeVariantCaution {
		t.Errorf("object badge: got %+v", router)
	}
	if router.Attrs["data-x"] != "y" {
		t.Errorf("attrs: got %v", router.Attrs)
	}

	if entry.Overrides["plugins/internal"] == nil || !entry.Overrides["plugins/internal"].Hidden {
		t.Error("hidden override not parsed")
	}
}

func TestLoadSidebarFile_UnknownKeyErrors(t *testing.T) {
	dir := t.TempDir()
	writeSidebarYAML(t, dir, `
docs:
  overrides:
    guide/advanced:
      lable: "typo"
`)
	if _, err := LoadSidebarFile(dir); err == nil {
		t.Fatal("expected strict decode error for unknown key")
	}
}

func TestLoadSidebarFile_ItemsParsed(t *testing.T) {
	dir := t.TempDir()
	writeSidebarYAML(t, dir, `
docs:
  items:
    - autogenerate: guide
    - label: GitHub
      url: https://github.com/example
      external: true
`)
	sf, err := LoadSidebarFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	items := sf["docs"].Items
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Autogenerate != "guide" {
		t.Errorf("autogenerate: got %q", items[0].Autogenerate)
	}
	if items[1].URL != "https://github.com/example" || !items[1].External {
		t.Errorf("external item: got %+v", items[1])
	}
}

func TestResolve_NoSidebarFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Resolve(ResolveOptions{ConfigPath: filepath.Join(dir, "sarde.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SidebarFile != nil {
		t.Errorf("SidebarFile must be nil without sidebar.yaml, got %v", cfg.SidebarFile)
	}
}

func TestResolve_LoadsSidebarFile(t *testing.T) {
	dir := t.TempDir()
	writeSidebarYAML(t, dir, `
docs:
  overrides:
    guide:
      label: "Guide"
`)
	cfg, err := Resolve(ResolveOptions{ConfigPath: filepath.Join(dir, "sarde.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SidebarFile == nil || cfg.SidebarFile["docs"] == nil {
		t.Fatal("expected sidebar.yaml to be loaded during Resolve")
	}
	if cfg.SidebarFile["docs"].Overrides["guide"].Label != "Guide" {
		t.Errorf("override label: got %+v", cfg.SidebarFile["docs"].Overrides["guide"])
	}
}
