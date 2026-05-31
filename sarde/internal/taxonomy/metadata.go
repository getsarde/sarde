package taxonomy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/frostybee/sarde/internal/consts"
)

// TermMetadata holds optional per-term configuration from data/<taxonomy>.yml.
type TermMetadata struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
	Color       string `yaml:"color"`
	Icon        string `yaml:"icon"`
	Hidden      bool   `yaml:"hidden"`
	Priority    int    `yaml:"priority"`
	Permalink   string `yaml:"permalink"`
	Difficulty  string `yaml:"difficulty"`
	ContentType string `yaml:"content_type"`
}

var validDifficulties = map[string]bool{
	"beginner": true, "intermediate": true, "advanced": true,
}

var validContentTypes = map[string]bool{
	"lecture": true, "lab": true, "assignment": true, "project": true,
	"reference": true, "tutorial": true, "assessment": true,
}

// termMetadataFile is the top-level YAML structure for data/<taxonomy>.yml.
// It supports a `defaults` key alongside per-term entries.
type termMetadataFile struct {
	Defaults *TermMetadata            `yaml:"defaults"`
	Terms    map[string]*TermMetadata `yaml:",inline"`
}

// LoadTermMetadata reads data/<taxonomyName>.yml from the project data dir.
// Returns a map of slug to TermMetadata. A missing file returns nil, nil.
func LoadTermMetadata(projectDir, taxonomyName string) (map[string]TermMetadata, error) {
	path := filepath.Join(projectDir, consts.DirData, taxonomyName+".yml")
	return loadTermMetadataFile(path)
}

// LoadTermMetadataForLang loads term metadata with per-language overlay.
// Reads the base data/<name>.yml, then overlays data/<name>.<lang>.yml
// (per-key merge, overlay wins). lang=="" returns base only.
func LoadTermMetadataForLang(projectDir, taxonomyName, lang string) (map[string]TermMetadata, error) {
	base, err := LoadTermMetadata(projectDir, taxonomyName)
	if err != nil {
		return nil, err
	}
	if lang == "" {
		return base, nil
	}

	overlayPath := filepath.Join(projectDir, consts.DirData, taxonomyName+"."+lang+".yml")
	overlay, err := loadTermMetadataFile(overlayPath)
	if err != nil {
		return nil, err
	}
	if overlay == nil {
		return base, nil
	}
	if base == nil {
		return overlay, nil
	}

	for slug, om := range overlay {
		bm, exists := base[slug]
		if !exists {
			base[slug] = om
			continue
		}
		if om.Label != "" {
			bm.Label = om.Label
		}
		if om.Description != "" {
			bm.Description = om.Description
		}
		if om.Color != "" {
			bm.Color = om.Color
		}
		if om.Icon != "" {
			bm.Icon = om.Icon
		}
		if om.Hidden {
			bm.Hidden = om.Hidden
		}
		if om.Priority != 0 {
			bm.Priority = om.Priority
		}
		if om.Permalink != "" {
			bm.Permalink = om.Permalink
		}
		if om.Difficulty != "" {
			bm.Difficulty = om.Difficulty
		}
		if om.ContentType != "" {
			bm.ContentType = om.ContentType
		}
		base[slug] = bm
	}
	return base, nil
}

func loadTermMetadataFile(path string) (map[string]TermMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var file termMetadataFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		// Fall back to the flat map format for backward compatibility.
		var flat map[string]TermMetadata
		if err2 := yaml.Unmarshal(data, &flat); err2 != nil {
			return nil, err
		}
		return flat, nil
	}

	result := make(map[string]TermMetadata, len(file.Terms))
	for slug, tm := range file.Terms {
		if slug == "defaults" {
			continue
		}
		if tm == nil {
			continue
		}
		m := *tm
		if err := validateTermMetadata(slug, &m); err != nil {
			return nil, err
		}
		if file.Defaults != nil {
			applyDefaults(&m, file.Defaults)
		}
		result[slug] = m
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func applyDefaults(m *TermMetadata, defaults *TermMetadata) {
	if m.Color == "" && defaults.Color != "" {
		m.Color = defaults.Color
	}
	if m.Priority == 0 && defaults.Priority != 0 {
		m.Priority = defaults.Priority
	}
	if m.Difficulty == "" && defaults.Difficulty != "" {
		m.Difficulty = defaults.Difficulty
	}
	if m.ContentType == "" && defaults.ContentType != "" {
		m.ContentType = defaults.ContentType
	}
}

func validateTermMetadata(slug string, m *TermMetadata) error {
	if m.Difficulty != "" && !validDifficulties[m.Difficulty] {
		return fmt.Errorf("tag %q: invalid difficulty %q (must be beginner, intermediate, or advanced)", slug, m.Difficulty)
	}
	if m.ContentType != "" && !validContentTypes[m.ContentType] {
		return fmt.Errorf("tag %q: invalid content_type %q (must be lecture, lab, assignment, project, reference, tutorial, or assessment)", slug, m.ContentType)
	}
	return nil
}
