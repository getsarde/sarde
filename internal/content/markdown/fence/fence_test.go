package fence

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		line string
		want Fence
	}{
		{":::", Fence{Kind: Close, Colons: 3}},
		{"::::", Fence{Kind: Close, Colons: 4}},
		{":::/note", Fence{Kind: Close, Name: "note", Colons: 3}},
		{"::::/card-grid", Fence{Kind: Close, Name: "card-grid", Colons: 4}},
		{":::/NOTE", Fence{Kind: Close, Name: "note", Colons: 3}},
		{":::/ note", Fence{Kind: Malformed, Name: "note", Colons: 3}},
		{":::/   ", Fence{Kind: Malformed, Colons: 3}},
		{":::/note x", Fence{Kind: Malformed, Name: "note", Colons: 3}},
		{":::/", Fence{Kind: Malformed, Colons: 3}},
		{":::note", Fence{Kind: Open, Name: "note", Colons: 3}},
		{":::note[Title] icon=flame", Fence{Kind: Open, Name: "note", Colons: 3}},
		{"::: note", Fence{Kind: Open, Name: "note", Colons: 3}},
		{":::Terminal", Fence{Kind: Open, Name: "terminal", Colons: 3}},
		{":::card-grid(cols=3)", Fence{Kind: Open, Name: "card-grid", Colons: 3}},
		{":::tab{label=\"Go\"}", Fence{Kind: Open, Name: "tab", Colons: 3}},
		{":::-x", Fence{Kind: None, Colons: 3}},
		{"::kbd[Ctrl]", Fence{}},
		{"text", Fence{}},
		{"", Fence{}},
	}
	for _, tt := range tests {
		if got := Classify(tt.line); got != tt.want {
			t.Errorf("Classify(%q) = %+v, want %+v", tt.line, got, tt.want)
		}
	}
}

func TestCloses(t *testing.T) {
	yes := [][2]string{
		{"tabs", "tabs"}, {"NOTE", "note"}, {"note", "NOTE"},
		{"aside", "note"}, {"aside", "gh-caution"}, {"filetree", "file-tree"},
	}
	no := [][2]string{
		{"column", "columns"}, {"columns", "column"}, {"tip", "note"},
		{"file-tree", "filetree"}, {"aside", "tabs"}, {"", "note"},
	}
	for _, p := range yes {
		if !Closes(p[0], p[1]) {
			t.Errorf("Closes(%q, %q) = false, want true", p[0], p[1])
		}
	}
	for _, p := range no {
		if Closes(p[0], p[1]) {
			t.Errorf("Closes(%q, %q) = true, want false", p[0], p[1])
		}
	}
}

func TestFindClosable(t *testing.T) {
	names := []string{"details", "note", "details", "tip"}
	if got := FindClosable(names, "details"); got != 2 {
		t.Errorf("innermost details: got %d, want 2", got)
	}
	if got := FindClosable(names, "aside"); got != 3 {
		t.Errorf("aside alias picks innermost aside: got %d, want 3", got)
	}
	if got := FindClosable(names, "tabs"); got != -1 {
		t.Errorf("no match: got %d, want -1", got)
	}
	if got := FindClosable(nil, "tabs"); got != -1 {
		t.Errorf("empty: got %d, want -1", got)
	}
}

func TestCodeFence(t *testing.T) {
	var c CodeFence
	feed := func(lines ...string) []bool {
		out := make([]bool, len(lines))
		for i, l := range lines {
			out[i] = c.Feed(l)
		}
		return out
	}
	got := feed("```go", ":::note", "```", ":::note")
	want := []bool{true, true, true, false}
	if !equal(got, want) {
		t.Errorf("backtick fence: got %v, want %v", got, want)
	}

	c = CodeFence{}
	got = feed("````md", "```", ":::note", "````", "after")
	want = []bool{true, true, true, true, false}
	if !equal(got, want) {
		t.Errorf("shorter inner fence does not close: got %v, want %v", got, want)
	}

	c = CodeFence{}
	got = feed("~~~", "```", "~~~~", "after")
	want = []bool{true, true, true, false}
	if !equal(got, want) {
		t.Errorf("tilde fence closed only by tildes: got %v, want %v", got, want)
	}

	c = CodeFence{}
	got = feed("```", "``` trailing", "```")
	want = []bool{true, true, true}
	if !equal(got, want) || c.Inside() {
		t.Errorf("closer must be alone on its line: got %v inside=%v", got, c.Inside())
	}

	c = CodeFence{}
	feed("```", "never closed")
	if !c.Inside() {
		t.Error("unclosed fence should stay inside")
	}
}

func equal(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
