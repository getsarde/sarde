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
	want := "unclosed block ':::details' (opened at line 2)"
	if diags[0].Message != want {
		t.Errorf("expected message %q, got %q", want, diags[0].Message)
	}
}

// An outer named closer over an unclosed inner block is valid recovery: the
// renderer closes both. The checker reports the inner block as implicitly
// closed, and the trailing bare ::: is then an orphan.
func TestCheck_NamedCloseRecoversUnclosedInner(t *testing.T) {
	content := []byte(`
:::accordion
:::details[Item 1]
Content
:::/accordion
:::
`)
	diags := Check("test.md", content, 0)
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d: %+v", len(diags), diags)
	}
	want := "unclosed block ':::details' (opened at line 3), closed implicitly by ':::/accordion' at line 5"
	if diags[0].Level != "warning" || diags[0].Tag != "details" || diags[0].Line != 3 || diags[0].Message != want {
		t.Errorf("recovery warning = %+v, want %q at line 3", diags[0], want)
	}
	if diags[1].Level != "error" || diags[1].Line != 6 || diags[1].Message != "orphaned closing tag with no matching opener" {
		t.Errorf("trailing bare ::: should be an orphan error at line 6, got %+v", diags[1])
	}
}

// A named closer that matches no open block renders as text, so the block
// stays open: one mismatch error, then an unclosed warning at EOF unless a
// later fence closes it.
func TestCheck_MismatchedExplicitClose(t *testing.T) {
	content := []byte(`
:::note
Content
:::/tip
:::
`)
	diags := Check("test.md", content, 0)
	if len(diags) != 1 {
		t.Fatalf("expected exactly 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	want := "mismatched closing tag ':::/tip', expected ':::/note' (opened at line 2)"
	if diags[0].Level != "error" || diags[0].Tag != "tip" || diags[0].Line != 4 || diags[0].Message != want {
		t.Errorf("got %+v, want %q", diags[0], want)
	}

	// Without the bare closer the note is reported unclosed too.
	diags = Check("test.md", []byte(":::note\nContent\n:::/tip\n"), 0)
	if len(diags) != 2 || diags[1].Message != "unclosed block ':::note' (opened at line 1)" {
		t.Errorf("mismatch must not pop the block: %+v", diags)
	}
}

func TestCheck_CloserAliasesAndCase(t *testing.T) {
	for _, content := range []string{
		":::note\nx\n:::/aside\n",
		":::gh-tip\nx\n:::/aside\n",
		":::note\nx\n:::/NOTE\n",
		":::Note\nx\n:::/note\n",
		":::file-tree\n- a\n:::/filetree\n",
		":::file-tree\n- a\n:::/FileTree\n",
	} {
		if diags := Check("test.md", []byte(content), 0); len(diags) != 0 {
			t.Errorf("%q: expected 0 diagnostics, got %+v", content, diags)
		}
	}
}

func TestCheck_MalformedCloser(t *testing.T) {
	diags := Check("test.md", []byte(":::note\nx\n:::/ note\n:::/note extra\n:::\n"), 0)
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d: %+v", len(diags), diags)
	}
	if diags[0].Line != 3 || diags[0].Tag != "note" || diags[0].Level != "error" ||
		diags[0].Message != "malformed closing fence ':::/ note': write ':::/note' with nothing after the name" {
		t.Errorf("got %+v", diags[0])
	}
	if diags[1].Line != 4 || diags[1].Message != "malformed closing fence ':::/note extra': write ':::/note' with nothing after the name" {
		t.Errorf("got %+v", diags[1])
	}
}

func TestCheck_CodeFenceLengthAndCharacter(t *testing.T) {
	// A shorter inner fence does not close a longer outer one, and a tilde
	// fence is not closed by backticks.
	for _, content := range []string{
		"````md\n```\n:::details[Not real]\n````\n",
		"~~~\n```\n:::details[Not real]\n~~~\n",
	} {
		if diags := Check("test.md", []byte(content), 0); len(diags) != 0 {
			t.Errorf("%q: expected 0 diagnostics, got %+v", content, diags)
		}
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
	want := "closing tag ':::/details' with no matching opener"
	if diags[0].Message != want {
		t.Errorf("expected message %q, got %q", want, diags[0].Message)
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
