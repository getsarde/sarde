package asset

import (
	"testing"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

func TestGetResource(t *testing.T) {
	resources := []engine.Resource{
		{Name: "hero.jpg", MediaType: "image/jpeg"},
		{Name: "doc.pdf", MediaType: "application/pdf"},
		{Name: "icon.png", MediaType: "image/png"},
	}

	r := GetResource(resources, "doc.pdf")
	if r == nil {
		t.Fatal("expected to find doc.pdf")
	}
	if r.Name != "doc.pdf" {
		t.Errorf("Name = %q, want doc.pdf", r.Name)
	}

	r = GetResource(resources, "missing.txt")
	if r != nil {
		t.Error("expected nil for missing resource")
	}
}

func TestMatchResources(t *testing.T) {
	resources := []engine.Resource{
		{Name: "hero.jpg"},
		{Name: "thumb.jpg"},
		{Name: "icon.png"},
		{Name: "doc.pdf"},
	}

	matched := MatchResources(resources, "*.jpg")
	if len(matched) != 2 {
		t.Errorf("len = %d, want 2", len(matched))
	}

	matched = MatchResources(resources, "*.png")
	if len(matched) != 1 {
		t.Errorf("len = %d, want 1", len(matched))
	}

	matched = MatchResources(resources, "*.txt")
	if len(matched) != 0 {
		t.Errorf("len = %d, want 0", len(matched))
	}
}

func TestResourcesByType(t *testing.T) {
	resources := []engine.Resource{
		{Name: "hero.jpg", MediaType: "image/jpeg"},
		{Name: "icon.png", MediaType: "image/png"},
		{Name: "doc.pdf", MediaType: "application/pdf"},
		{Name: "style.css", MediaType: "text/css"},
	}

	images := ResourcesByType(resources, "image")
	if len(images) != 2 {
		t.Errorf("image count = %d, want 2", len(images))
	}

	apps := ResourcesByType(resources, "application")
	if len(apps) != 1 {
		t.Errorf("application count = %d, want 1", len(apps))
	}

	texts := ResourcesByType(resources, "text")
	if len(texts) != 1 {
		t.Errorf("text count = %d, want 1", len(texts))
	}
}

func TestEnhancePageResources_SetsMetadata(t *testing.T) {
	page := &engine.Page{
		FilePath:     "/project/content/blog/my-post/index.md",
		RelPermalink: "/blog/my-post/",
		Resources: []engine.Resource{
			{Name: "doc.pdf"},
			{Name: "data.json"},
		},
	}

	enhancer := &ResourceEnhancer{DevMode: true}
	err := enhancer.EnhancePageResources(page)
	if err != nil {
		t.Fatalf("EnhancePageResources failed: %v", err)
	}

	// Check doc.pdf
	pdf := page.Resources[0]
	if pdf.Title != "doc" {
		t.Errorf("Title = %q, want doc", pdf.Title)
	}
	if pdf.MediaType != "application/pdf" {
		t.Errorf("MediaType = %q, want application/pdf", pdf.MediaType)
	}
	if pdf.RelPermalink != "/blog/my-post/doc.pdf" {
		t.Errorf("RelPermalink = %q, want /blog/my-post/doc.pdf", pdf.RelPermalink)
	}

	// Check data.json
	json := page.Resources[1]
	if json.Title != "data" {
		t.Errorf("Title = %q, want data", json.Title)
	}
	if json.MediaType != "application/json" {
		t.Errorf("MediaType = %q, want application/json", json.MediaType)
	}
}

func TestEnhancePageResources_Empty(t *testing.T) {
	page := &engine.Page{
		FilePath:     "/project/content/about.md",
		RelPermalink: "/about/",
	}

	enhancer := &ResourceEnhancer{}
	err := enhancer.EnhancePageResources(page)
	if err != nil {
		t.Fatalf("EnhancePageResources failed: %v", err)
	}
}
