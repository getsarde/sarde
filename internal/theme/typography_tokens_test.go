package theme

import (
	"strings"
	"testing"
)

// font-heading and text-scale are ordinary tokens: overrides for them pass
// validation and reach the generated :root block.
func TestTypographyTokens_ReachStyleTag(t *testing.T) {
	overrides := map[string]string{"font-heading": "Georgia, serif", "text-scale": "1.25"}
	if err := ValidateOverrides("theme.overrides", overrides, KnownTokens()); err != nil {
		t.Fatalf("overrides rejected: %v", err)
	}
	tokens := ResolveTokens(DefaultTokens(), nil, "", overrides)
	tag := string(GenerateStyleTag(tokens, nil))
	for _, want := range []string{"--sd-font-heading: Georgia, serif;", "--sd-text-scale: 1.25;"} {
		if !strings.Contains(tag, want) {
			t.Errorf("style tag missing %q:\n%s", want, tag)
		}
	}
}
