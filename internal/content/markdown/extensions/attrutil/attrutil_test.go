package attrutil

import "testing"

func TestParse_DoubleQuoted(t *testing.T) {
	attrs := Parse(`icon="star" href="/path"`)
	assertVal(t, attrs, "icon", "star")
	assertVal(t, attrs, "href", "/path")
}

func TestParse_SingleQuoted(t *testing.T) {
	attrs := Parse(`icon='star' href='/path'`)
	assertVal(t, attrs, "icon", "star")
	assertVal(t, attrs, "href", "/path")
}

func TestParse_BareFlags(t *testing.T) {
	attrs := Parse("stagger independent")
	assertHas(t, attrs, "stagger")
	assertHas(t, attrs, "independent")
	assertVal(t, attrs, "stagger", "")
}

func TestParse_MixedAttrsAndFlags(t *testing.T) {
	attrs := Parse(`cols="3" stagger`)
	assertVal(t, attrs, "cols", "3")
	assertHas(t, attrs, "stagger")
}

func TestParse_HyphenatedKeys(t *testing.T) {
	attrs := Parse(`new-tab="true" icon-placement="start" no-icon="true"`)
	assertVal(t, attrs, "new-tab", "true")
	assertVal(t, attrs, "icon-placement", "start")
	assertVal(t, attrs, "no-icon", "true")
}

func TestParse_EmptyString(t *testing.T) {
	attrs := Parse("")
	if len(attrs) != 0 {
		t.Errorf("expected empty map, got %v", attrs)
	}
}

func TestParse_EmptyValue(t *testing.T) {
	attrs := Parse(`title=""`)
	assertVal(t, attrs, "title", "")
}

func TestParse_ComplexValues(t *testing.T) {
	attrs := Parse(`src="https://youtube.com/watch?v=abc" ratio="4:3" autoplay muted`)
	assertVal(t, attrs, "src", "https://youtube.com/watch?v=abc")
	assertVal(t, attrs, "ratio", "4:3")
	assertHas(t, attrs, "autoplay")
	assertHas(t, attrs, "muted")
}

func TestHas(t *testing.T) {
	attrs := Parse(`icon="star" stagger`)
	if !Has(attrs, "icon") {
		t.Error("expected Has(icon) = true")
	}
	if !Has(attrs, "stagger") {
		t.Error("expected Has(stagger) = true")
	}
	if Has(attrs, "missing") {
		t.Error("expected Has(missing) = false")
	}
}

func TestBool(t *testing.T) {
	attrs := Parse(`disabled="true" center="false" open`)
	if !Bool(attrs, "disabled") {
		t.Error("expected Bool(disabled) = true")
	}
	if Bool(attrs, "center") {
		t.Error("expected Bool(center) = false")
	}
	if Bool(attrs, "open") {
		t.Error("expected Bool(open) = false for bare flag")
	}
	if Bool(attrs, "missing") {
		t.Error("expected Bool(missing) = false")
	}
}

func assertVal(t *testing.T, attrs map[string]string, key, want string) {
	t.Helper()
	got, ok := attrs[key]
	if !ok {
		t.Errorf("key %q not found in attrs", key)
		return
	}
	if got != want {
		t.Errorf("attrs[%q] = %q, want %q", key, got, want)
	}
}

func assertHas(t *testing.T, attrs map[string]string, key string) {
	t.Helper()
	if !Has(attrs, key) {
		t.Errorf("expected key %q to be present", key)
	}
}
