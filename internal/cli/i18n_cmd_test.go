package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestI18nStrings_ListsEveryLayer(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "sarde.yaml"), []byte(
		"i18n:\n  default_language: en\n  languages:\n    fr: {weight: 2}\n    en: {weight: 1}\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "i18n"), 0o755)
	os.WriteFile(filepath.Join(dir, "i18n", "fr.yaml"), []byte("nav:\n  toc: Sur cette page\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "i18n", "de.yaml"), []byte("nav: [unclosed\n"), 0o644)

	out, err := captureStdout(t, func() error {
		rootCmd.SetArgs([]string{"i18n", "strings", dir})
		return rootCmd.Execute()
	})
	if err != nil {
		t.Fatalf("i18n strings: %v (%s)", err, out)
	}
	var got struct {
		OK              bool     `json:"ok"`
		DefaultLanguage string   `json:"defaultLanguage"`
		Languages       []string `json:"languages"`
		Layers          []struct {
			Source  string                       `json:"source"`
			Dir     string                       `json:"dir"`
			Strings map[string]map[string]string `json:"strings"`
		} `json:"layers"`
		Problems []struct {
			File string `json:"file"`
		} `json:"problems"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode %q: %v", out, err)
	}
	if !got.OK || got.DefaultLanguage != "en" || len(got.Languages) != 2 || got.Languages[0] != "en" {
		t.Errorf("header: %+v", got)
	}
	sources := map[string]map[string]map[string]string{}
	for _, l := range got.Layers {
		sources[l.Source] = l.Strings
	}
	if sources["builtin"]["en"]["nav.toc"] == "" {
		t.Error("the built-in English strings are missing")
	}
	if sources["project"]["fr"]["nav.toc"] != "Sur cette page" {
		t.Errorf("project fr: %v", sources["project"]["fr"])
	}
	if _, ok := sources["theme"]; !ok {
		t.Error("the default theme layer is missing")
	}
	if len(got.Problems) != 1 || got.Problems[0].File != "i18n/de.yaml" {
		t.Errorf("problems: %+v", got.Problems)
	}
}
