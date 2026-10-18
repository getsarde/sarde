package content

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Getting Started", "getting-started"},
		{"getting_started_guide", "getting-started-guide"},
		{"GettingStarted", "gettingstarted"},
		{"My (Awesome) Post", "my-awesome-post"},
		{"  leading--and--trailing  ", "leading-and-trailing"},
		{"Hello World!", "hello-world"},
		{"", ""},
		{"already-slugified", "already-slugified"},
		{"UPPER_CASE_SLUG", "upper-case-slug"},
	}
	for _, tt := range tests {
		got := Slugify(tt.input)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractNumericPrefix(t *testing.T) {
	tests := []struct {
		input     string
		wantW     int
		wantSlug  string
		wantFound bool
	}{
		{"01-intro", 1, "intro", true},
		{"02-getting-started", 2, "getting-started", true},
		{"03_overview", 3, "overview", true},
		{"10-chapter-two", 10, "chapter-two", true},
		{"99-appendix", 99, "appendix", true},
		{"001-basics", 1, "basics", true},
		{"getting-started", 0, "getting-started", false},
		{"intro", 0, "intro", false},
		{"0-empty", 0, "empty", true},
	}
	for _, tt := range tests {
		w, slug, found := ExtractNumericPrefix(tt.input)
		if w != tt.wantW || slug != tt.wantSlug || found != tt.wantFound {
			t.Errorf("ExtractNumericPrefix(%q) = (%d, %q, %v), want (%d, %q, %v)",
				tt.input, w, slug, found, tt.wantW, tt.wantSlug, tt.wantFound)
		}
	}
}

func TestExtractFirstH1(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"# Hello World\n\nSome content", "Hello World"},
		{"Some intro\n\n# The Title\n\nContent", "The Title"},
		{"## Not H1\n\nContent", ""},
		{"No headings at all", ""},
		{"# First\n# Second", "First"},
		{"#Not a heading", ""},
		{"# Trimmed  ", "Trimmed"},
	}
	for _, tt := range tests {
		got := ExtractFirstH1(tt.input)
		if got != tt.want {
			t.Errorf("ExtractFirstH1(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFilenameToTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"getting-started.md", "Getting Started"},
		{"01-intro.md", "Intro"},
		{"02-getting-started.md", "Getting Started"},
		{"my_awesome_post.md", "My Awesome Post"},
		{"simple.md", "Simple"},
	}
	for _, tt := range tests {
		got := FilenameToTitle(tt.input)
		if got != tt.want {
			t.Errorf("FilenameToTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFilenameSlug(t *testing.T) {
	tests := []struct {
		input      string
		wantSlug   string
		wantWeight int
	}{
		{"01-getting-started.md", "getting-started", 1},
		{"getting-started.md", "getting-started", 0},
		{"My Post.md", "my-post", 0},
		{"03_overview.md", "overview", 3},
	}
	for _, tt := range tests {
		slug, weight := FilenameSlug(tt.input)
		if slug != tt.wantSlug || weight != tt.wantWeight {
			t.Errorf("FilenameSlug(%q) = (%q, %d), want (%q, %d)",
				tt.input, slug, weight, tt.wantSlug, tt.wantWeight)
		}
	}
}

func TestSlugOrHash(t *testing.T) {
	if got := SlugOrHash("Getting Started"); got != "getting-started" {
		t.Errorf("SlugOrHash(ASCII) = %q", got)
	}
	a, b := SlugOrHash("日本語"), SlugOrHash("Привет")
	if len(a) != 8 || len(b) != 8 || a == b {
		t.Errorf("SlugOrHash(non-Latin) = %q, %q, want two distinct 8-hex hashes", a, b)
	}
	if SlugOrHash("日本語") != a {
		t.Error("SlugOrHash is not stable")
	}
}

func TestFileSlug(t *testing.T) {
	tests := []struct {
		name      string
		wantSlug  string
		wantOrder int
	}{
		{"hello-world", "hello-world", 0},
		{"03-setup", "setup", 3},
		{"2024-03-15-hello", "hello", 0},
		{"2024-01-15-02-intro", "intro", 2},
		{"2024-02-30-bad-date", "02-30-bad-date", 2024},
		{"01-日本語", "01", 1},
	}
	for _, tt := range tests {
		slug, order := FileSlug(tt.name)
		if slug != tt.wantSlug || order != tt.wantOrder {
			t.Errorf("FileSlug(%q) = (%q, %d), want (%q, %d)", tt.name, slug, order, tt.wantSlug, tt.wantOrder)
		}
	}
	if slug, _ := FileSlug("日本語"); slug != SlugOrHash("日本語") {
		t.Errorf("FileSlug(non-Latin) = %q, want the hash", slug)
	}
}
