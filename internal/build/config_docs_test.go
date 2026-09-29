package build

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// configDocsDir holds the Configuration reference pages, one topic per file.
const configDocsDir = "../../docs/content/docs/reference/configuration"

// completeExampleHeading introduces the example that sets every key.
const completeExampleHeading = "## Every key with its default"

var sardeYAMLBlock = regexp.MustCompile("(?s)```yaml title=\"sarde.yaml\"\n(.*?)```")

// TestConfigExamplesDocs_Resolve is a drift guard for examples.md: every
// sarde.yaml example must load the way a real build loads it, with unknown
// keys rejected and plugin names checked.
func TestConfigExamplesDocs_Resolve(t *testing.T) {
	doc := readConfigDoc(t, "examples.md")
	blocks := sardeYAMLBlock.FindAllStringSubmatchIndex(doc, -1)
	if len(blocks) == 0 {
		t.Fatal("examples.md has no ```yaml title=\"sarde.yaml\" blocks")
	}
	known := KnownPluginNames("")
	for _, b := range blocks {
		dir := t.TempDir()
		path := filepath.Join(dir, "sarde.yaml")
		if err := os.WriteFile(path, []byte(doc[b[2]:b[3]]), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := config.Resolve(config.ResolveOptions{
			ConfigPath:   path,
			EnvPrefix:    "SARDE_CONFIG_DOCS_TEST_UNSET",
			Strict:       true,
			KnownPlugins: known,
		})
		if err != nil {
			t.Errorf("example under %q does not resolve: %v", headingBefore(doc, b[0]), err)
		}
	}
}

// TestConfigExamplesDocs_CompleteMatchesDefaults checks that the "every key"
// example lists every embedded default with the same value, and every
// top-level key of SiteConfig.
func TestConfigExamplesDocs_CompleteMatchesDefaults(t *testing.T) {
	doc := readConfigDoc(t, "examples.md")
	idx := strings.Index(doc, completeExampleHeading)
	if idx < 0 {
		t.Fatalf("examples.md has no %q section", completeExampleHeading)
	}
	m := sardeYAMLBlock.FindStringSubmatch(doc[idx:])
	if m == nil {
		t.Fatalf("no sarde.yaml block after %q", completeExampleHeading)
	}
	var documented, defaults map[string]any
	if err := yaml.Unmarshal([]byte(m[1]), &documented); err != nil {
		t.Fatalf("complete example is not valid YAML: %v", err)
	}
	if err := yaml.Unmarshal(embedded.DefaultsYAML, &defaults); err != nil {
		t.Fatalf("embedded defaults: %v", err)
	}
	compareSubset(t, "", defaults, documented)

	for _, key := range topLevelConfigKeys() {
		if _, ok := documented[key]; !ok {
			t.Errorf("complete example is missing top-level key %q", key)
		}
	}
}

// TestConfigDocs_EveryKeyHasOnePage checks that each top-level SiteConfig key
// has a "## `key`" heading on exactly one Configuration topic page.
func TestConfigDocs_EveryKeyHasOnePage(t *testing.T) {
	entries, err := os.ReadDir(configDocsDir)
	if err != nil {
		t.Fatalf("reading %s: %v", configDocsDir, err)
	}
	pagesFor := map[string][]string{}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		doc := readConfigDoc(t, e.Name())
		for _, key := range topLevelConfigKeys() {
			if strings.Contains(doc, "\n## `"+key+"`\n") {
				pagesFor[key] = append(pagesFor[key], e.Name())
			}
		}
	}
	for _, key := range topLevelConfigKeys() {
		switch pages := pagesFor[key]; len(pages) {
		case 1:
		case 0:
			t.Errorf("top-level key %q has no \"## `%s`\" heading in %s", key, key, configDocsDir)
		default:
			t.Errorf("top-level key %q is documented on several pages: %v", key, pages)
		}
	}
}

// compareSubset reports every path in want that is missing from got or has a
// different value.
func compareSubset(t *testing.T, path string, want, got any) {
	t.Helper()
	wm, wantMap := want.(map[string]any)
	if wantMap && len(wm) > 0 {
		gm, ok := got.(map[string]any)
		if !ok {
			t.Errorf("%s: documented as %v, want a mapping", path, got)
			return
		}
		keys := make([]string, 0, len(wm))
		for k := range wm {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := k
			if path != "" {
				child = path + "." + k
			}
			gv, ok := gm[k]
			if !ok {
				t.Errorf("%s: missing from the complete example (default %v)", child, wm[k])
				continue
			}
			compareSubset(t, child, wm[k], gv)
		}
		return
	}
	if !reflect.DeepEqual(normalizeEmpty(want), normalizeEmpty(got)) {
		t.Errorf("%s: documented as %v, default is %v", path, got, want)
	}
}

// normalizeEmpty treats nil, empty lists and empty maps as equal, since
// `[]`, `{}` and an omitted value all decode to "nothing set".
func normalizeEmpty(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case []any:
		if len(x) == 0 {
			return nil
		}
	case map[string]any:
		if len(x) == 0 {
			return nil
		}
	}
	return v
}

// topLevelConfigKeys returns the yaml keys of SiteConfig, skipping fields that
// never come from sarde.yaml.
func topLevelConfigKeys() []string {
	typ := reflect.TypeOf(config.SiteConfig{})
	var keys []string
	for i := 0; i < typ.NumField(); i++ {
		tag := strings.Split(typ.Field(i).Tag.Get("yaml"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		keys = append(keys, tag)
	}
	return keys
}

func readConfigDoc(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(configDocsDir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

// headingBefore returns the last "## " heading before offset, naming the
// example a failure belongs to.
func headingBefore(doc string, offset int) string {
	i := strings.LastIndex(doc[:offset], "\n## ")
	if i < 0 {
		return "(no heading)"
	}
	line := doc[i+1:]
	if j := strings.IndexByte(line, '\n'); j >= 0 {
		line = line[:j]
	}
	return line
}
