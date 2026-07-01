package syntax

import (
	"testing"
)

func TestCheck_BalancedBlocks(t *testing.T) {
	content := []byte(`
:::details[Summary]
Some content
:::
`)
	diags := Check("test.md", content, 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %+v", len(diags), diags)
	}
}

func TestCheck_NestedBalanced(t *testing.T) {
	content := []byte(`
:::accordion
:::details[Item 1]
Content
:::
:::details[Item 2]
Content
:::
:::
`)
	diags := Check("test.md", content, 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %+v", len(diags), diags)
	}
}

func TestCheck_ExplicitCloseBalanced(t *testing.T) {
	content := []byte(`
:::accordion
:::details[Item 1]
Content
:::/details
:::/accordion
`)
	diags := Check("test.md", content, 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %+v", len(diags), diags)
	}
}

func TestCheck_UnclosedBlock(t *testing.T) {
	content := []byte(`
:::details[Summary]
Some content
`)
	diags := Check("test.md", content, 0)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	if diags[0].Level != "warning" {
		t.Errorf("expected warning, got %s", diags[0].Level)
	}
	if diags[0].Tag != "details" {
		t.Errorf("expected tag 'details', got %s", diags[0].Tag)
	}
}

func TestCheck_MismatchedExplicitClose(t *testing.T) {
	content := []byte(`
:::accordion
:::details[Item 1]
Content
:::/accordion
:::
`)
	diags := Check("test.md", content, 0)
	if len(diags) == 0 {
		t.Fatal("expected at least 1 diagnostic for mismatched close")
	}
	found := false
	for _, d := range diags {
		if d.Level == "error" && d.Tag == "accordion" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error diagnostic for mismatched 'accordion' close, got: %+v", diags)
	}
}

func TestCheck_OrphanedClose(t *testing.T) {
	content := []byte(`
Some text
:::
`)
	diags := Check("test.md", content, 0)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	if diags[0].Level != "error" {
		t.Errorf("expected error, got %s", diags[0].Level)
	}
}

func TestCheck_OrphanedExplicitClose(t *testing.T) {
	content := []byte(`
:::/details
`)
	diags := Check("test.md", content, 0)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	if diags[0].Level != "error" {
		t.Errorf("expected error, got %s", diags[0].Level)
	}
}

func TestCheck_SkipsCodeFences(t *testing.T) {
	content := []byte("```markdown\n:::details[Not real]\nSome content\n```\n")
	diags := Check("test.md", content, 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics (inside code fence), got %d: %+v", len(diags), diags)
	}
}

func TestCheck_SkipsTildeCodeFences(t *testing.T) {
	content := []byte("~~~\n:::details[Not real]\n~~~\n")
	diags := Check("test.md", content, 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics (inside tilde fence), got %d: %+v", len(diags), diags)
	}
}

func TestCheck_MultipleUnclosed(t *testing.T) {
	content := []byte(`
:::accordion
:::details[Item 1]
Content
`)
	diags := Check("test.md", content, 0)
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d: %+v", len(diags), diags)
	}
}

func TestCheck_FourColonOpener(t *testing.T) {
	content := []byte(`
::::accordion
:::details[Item]
Content
:::
::::
`)
	diags := Check("test.md", content, 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %+v", len(diags), diags)
	}
}

func TestCheck_AttributesIgnored(t *testing.T) {
	content := []byte(`
:::details(open)[Summary text]
Content
:::
`)
	diags := Check("test.md", content, 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %+v", len(diags), diags)
	}
}

func TestCheck_BareSlashNoPanic(t *testing.T) {
	content := []byte(":::/\n")
	diags := Check("test.md", content, 0)
	_ = diags // must not panic
}

func TestCheck_SlashWithSpaces(t *testing.T) {
	content := []byte(":::/   \n")
	diags := Check("test.md", content, 0)
	_ = diags
}

func TestCheck_OnlyColons(t *testing.T) {
	content := []byte("::::\n:::::\n::::::\n")
	diags := Check("test.md", content, 0)
	_ = diags
}

func TestCheck_EmptyFile(t *testing.T) {
	diags := Check("test.md", []byte(""), 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for empty file, got %d", len(diags))
	}
}

func TestCheck_OnlyNewlines(t *testing.T) {
	diags := Check("test.md", []byte("\n\n\n"), 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(diags))
	}
}

func TestCheck_ColonsWithWhitespace(t *testing.T) {
	content := []byte(":::   \n")
	diags := Check("test.md", content, 0)
	_ = diags
}

func TestCheck_IndentedFencedBlock(t *testing.T) {
	content := []byte("  :::details[Test]\n  Content\n  :::\n")
	diags := Check("test.md", content, 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %+v", len(diags), diags)
	}
}

func TestCheck_TabIndentedBlock(t *testing.T) {
	content := []byte("\t:::details[Test]\n\tContent\n\t:::\n")
	diags := Check("test.md", content, 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %+v", len(diags), diags)
	}
}

func TestCheck_WindowsLineEndings(t *testing.T) {
	content := []byte(":::details[Test]\r\nContent\r\n:::\r\n")
	diags := Check("test.md", content, 0)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics, got %d: %+v", len(diags), diags)
	}
}

func TestCheck_SingleColonLineNoPanic(t *testing.T) {
	content := []byte(":\n::\n:::\n")
	diags := Check("test.md", content, 0)
	_ = diags
}

func TestCheck_SlashNoTagIsOrphan(t *testing.T) {
	content := []byte(":::details[X]\n:::/\n")
	diags := Check("test.md", content, 0)
	// :::/  with no tag name should be skipped, leaving details unclosed
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic (unclosed details), got %d: %+v", len(diags), diags)
	}
}
