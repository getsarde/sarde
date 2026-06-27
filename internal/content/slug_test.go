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
