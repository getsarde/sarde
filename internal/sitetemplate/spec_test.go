package sitetemplate

import (
	"errors"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/download"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want Spec
	}{
		{"course", Spec{Owner: "getsarde", Repo: "sarde-templates", Subpath: "course", Official: true}},
		{"course#main", Spec{Owner: "getsarde", Repo: "sarde-templates", Subpath: "course", Ref: "main", Official: true}},
		{"course#release/1.5", Spec{Owner: "getsarde", Repo: "sarde-templates", Subpath: "course", Ref: "release/1.5", Official: true}},
		{"getsarde/sarde-templates/course", Spec{Owner: "getsarde", Repo: "sarde-templates", Subpath: "course"}},
		{"acme/site", Spec{Owner: "acme", Repo: "site"}},
		{"acme/site.git/sub", Spec{Owner: "acme", Repo: "site", Subpath: "sub"}},
		{"acme/site#v2", Spec{Owner: "acme", Repo: "site", Ref: "v2"}},
		{"https://github.com/acme/site", Spec{Owner: "acme", Repo: "site"}},
		{"github.com/acme/site/tree/dev/sub/dir", Spec{Owner: "acme", Repo: "site", Subpath: "sub/dir", Ref: "dev"}},
		{"github.com/acme/site/tree/dev#v2", Spec{Owner: "acme", Repo: "site", Ref: "v2"}},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.in, err)
			continue
		}
		tt.want.Input = tt.in
		if got != tt.want {
			t.Errorf("Parse(%q)\n got %+v\nwant %+v", tt.in, got, tt.want)
		}
	}
}

func TestParse_Invalid(t *testing.T) {
	for _, in := range []string{"", "acme/", "/site", "acme/site/../x", "course#", "course# x", "acme/site#-bad", "a/b c"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) should fail", in)
		}
	}
}

func TestParse_UnknownNameIsTypedAndOffline(t *testing.T) {
	_, err := Parse("bogus")
	var unknown *UnknownTemplateError
	if !errors.As(err, &unknown) || unknown.Name != "bogus" {
		t.Fatalf("expected UnknownTemplateError, got %v", err)
	}
	if !strings.Contains(err.Error(), `unknown template "bogus"`) || !strings.Contains(err.Error(), "course") {
		t.Errorf("message should name the template and list the registry: %v", err)
	}
}

func TestDisplayAndCloneURL(t *testing.T) {
	s, _ := Parse("course#v1.5")
	if s.Display() != "course#v1.5" {
		t.Errorf("Display = %q", s.Display())
	}
	s, _ = Parse("acme/site/sub")
	if s.Display() != "acme/site/sub" || s.CloneURL() != "https://github.com/acme/site.git" {
		t.Errorf("Display = %q, CloneURL = %q", s.Display(), s.CloneURL())
	}
}

func TestResolveRef(t *testing.T) {
	official, _ := Parse("course")
	third, _ := Parse("acme/site")
	explicit, _ := Parse("course#v1.4")
	tests := []struct {
		name    string
		spec    Spec
		version string
		want    Resolved
	}{
		{"release pins to minor", official, "1.4.0", Resolved{Ref: "v1.4", Kind: download.RefTag, Pinned: true}},
		{"leading v accepted", official, "v1.4.2", Resolved{Ref: "v1.4", Kind: download.RefTag, Pinned: true}},
		{"rc pins too", official, "1.5.0-rc.1", Resolved{Ref: "v1.5", Kind: download.RefTag, Pinned: true}},
		{"dev build", official, "dev", Resolved{Ref: "main", Kind: download.RefBranch}},
		{"git describe", official, "v1.4.0-12-gabc1234-dirty", Resolved{Ref: "main", Kind: download.RefBranch}},
		{"snapshot", official, "1.5.0-SNAPSHOT-abc1234", Resolved{Ref: "main", Kind: download.RefBranch}},
		{"third party", third, "1.4.0", Resolved{Ref: "main", Kind: download.RefBranch}},
		{"explicit ref", explicit, "1.4.0", Resolved{Ref: "v1.4", Kind: download.RefAny}},
	}
	for _, tt := range tests {
		if got := ResolveRef(tt.spec, tt.version); got != tt.want {
			t.Errorf("%s: got %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
