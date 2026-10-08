package webfonts

import (
	"reflect"
	"strings"
	"testing"
)

func TestLookup_IgnoresCaseAndReturnsTheCanonicalName(t *testing.T) {
	for _, tc := range []struct{ in, name, category string }{
		{"Plus Jakarta Sans", "Plus Jakarta Sans", "sans-serif"},
		{"plus jakarta sans", "Plus Jakarta Sans", "sans-serif"},
		{"  MERRIWEATHER ", "Merriweather", "serif"},
		{"fira code", "Fira Code", "monospace"},
		{"Source Sans 3", "Source Sans 3", "sans-serif"},
	} {
		name, category, ok := Lookup(tc.in)
		if !ok || name != tc.name || category != tc.category {
			t.Errorf("Lookup(%q) = %q, %q, %v; want %q, %q, true", tc.in, name, category, ok, tc.name, tc.category)
		}
	}
	for _, missing := range []string{"Segoe UI", "Georgia", "Menlo", ""} {
		if _, _, ok := Lookup(missing); ok {
			t.Errorf("Lookup(%q) found a family; it is not on Google Fonts", missing)
		}
	}
}

func TestCatalog_IsLargeAndClean(t *testing.T) {
	if n := len(catalog()); n < 1500 {
		t.Fatalf("catalog has %d families; families.tsv looks truncated", n)
	}
	valid := map[string]bool{"sans-serif": true, "serif": true, "display": true, "handwriting": true, "monospace": true}
	for _, e := range catalog() {
		if !valid[e.category] {
			t.Errorf("%q has category %q", e.name, e.category)
		}
		if strings.ContainsAny(e.name, "&<>\"'|:,") {
			t.Errorf("%q would need URL or HTML escaping", e.name)
		}
	}
}

func TestPrimaryFamily(t *testing.T) {
	for _, tc := range []struct{ stack, want string }{
		{"'Plus Jakarta Sans', system-ui, sans-serif", "Plus Jakarta Sans"},
		{`"Merriweather", 'Georgia', serif`, "Merriweather"},
		{"Plus Jakarta Sans", "Plus Jakarta Sans"},
		{"  Fira   Code , monospace", "Fira Code"},
		{"system-ui, -apple-system, sans-serif", ""},
		{"-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif", "Segoe UI"},
		{"ui-monospace, 'JetBrains Mono', monospace", "JetBrains Mono"},
		{"'Comma, Inside', serif", "Comma, Inside"},
		{"", ""},
	} {
		if got := PrimaryFamily(tc.stack); got != tc.want {
			t.Errorf("PrimaryFamily(%q) = %q, want %q", tc.stack, got, tc.want)
		}
	}
}

func TestStackHas(t *testing.T) {
	for _, tc := range []struct {
		stack, family string
		want          bool
	}{
		{"'Inter', system-ui, sans-serif", "Inter", true},
		{"'Inter', system-ui, sans-serif", "inter", true},
		{`"Inter Tight", sans-serif`, "Inter", false},
		{"Interstate, sans-serif", "Inter", false},
		{"system-ui, 'Inter'", "Inter", true},
		{"'Comma, Inside', serif", "Comma, Inside", true},
		{"", "Inter", false},
	} {
		if got := StackHas(tc.stack, tc.family); got != tc.want {
			t.Errorf("StackHas(%q, %q) = %v, want %v", tc.stack, tc.family, got, tc.want)
		}
	}
}

func TestIsGeneric(t *testing.T) {
	for _, g := range []string{"serif", "SANS-SERIF", "system-ui", "-apple-system", "BlinkMacSystemFont", "ui-monospace"} {
		if !IsGeneric(g) {
			t.Errorf("IsGeneric(%q) = false", g)
		}
	}
	for _, f := range []string{"Inter", "Georgia", "Plus Jakarta Sans"} {
		if IsGeneric(f) {
			t.Errorf("IsGeneric(%q) = true", f)
		}
	}
}

func TestFamilies(t *testing.T) {
	light := map[string]string{
		"font-sans":    "'Plus Jakarta Sans', system-ui, sans-serif",
		"font-heading": "plus jakarta sans",         // same family, other case: once
		"font-mono":    "'Fira Code', ui-monospace", // a second family
	}
	dark := map[string]string{
		"font-sans": "Merriweather, serif", // dark tokens count too
		"font-mono": "'JetBrains Mono', monospace",
	}
	got := Families(light, dark)
	want := []string{"Plus Jakarta Sans", "Fira Code", "Merriweather"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Families = %q, want %q", got, want)
	}
}

func TestFamilies_SkipsBundledSystemAndUnknownFonts(t *testing.T) {
	tokens := map[string]string{
		"font-sans":    "'Inter', 'Inter Fallback', system-ui, sans-serif",
		"font-heading": "'Segoe UI', Roboto, sans-serif", // primary is not a web font
		"font-mono":    "'JetBrains Mono', ui-monospace, monospace",
	}
	if got := Families(tokens, nil); len(got) != 0 {
		t.Fatalf("Families = %q, want none", got)
	}
}

func TestStylesheetURL(t *testing.T) {
	fams := []string{"Plus Jakarta Sans", "Fira Code"}
	want := "https://fonts.googleapis.com/css?family=Plus+Jakarta+Sans:" + Weights +
		"|Fira+Code:" + Weights + "&display=swap"
	if got := StylesheetURL("google", fams); got != want {
		t.Errorf("google URL = %q\nwant %q", got, want)
	}
	wantBunny := strings.Replace(want, "https://fonts.googleapis.com/", "https://fonts.bunny.net/", 1)
	if got := StylesheetURL("bunny", fams); got != wantBunny {
		t.Errorf("bunny URL = %q\nwant %q", got, wantBunny)
	}
	if got := StylesheetURL("google", nil); got != "" {
		t.Errorf("no families: %q, want empty", got)
	}
	if got := StylesheetURL("adobe", fams); got != "" {
		t.Errorf("unknown provider: %q, want empty", got)
	}
}

func TestHeadLinks(t *testing.T) {
	google := HeadLinks("google", []string{"Lora"})
	for _, want := range []string{
		`<link rel="preconnect" href="https://fonts.googleapis.com">`,
		`<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>`,
		`<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Lora:` + Weights + `&amp;display=swap">`,
	} {
		if !strings.Contains(google, want) {
			t.Errorf("google links missing %s\n%s", want, google)
		}
	}
	bunny := HeadLinks("bunny", []string{"Lora"})
	if !strings.Contains(bunny, `<link rel="preconnect" href="https://fonts.bunny.net" crossorigin>`) ||
		strings.Contains(bunny, "googleapis") || strings.Contains(bunny, "gstatic") {
		t.Errorf("bunny links wrong:\n%s", bunny)
	}
	if HeadLinks("google", nil) != "" || HeadLinks("", []string{"Lora"}) != "" {
		t.Error("HeadLinks should be empty with no families or no provider")
	}
}
