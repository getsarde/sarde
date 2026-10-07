package sitetemplate

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/getsarde/sarde/internal/download"
)

// Spec is a parsed --template value.
type Spec struct {
	Input    string // as typed, for messages
	Owner    string
	Repo     string
	Subpath  string // slash-separated folder inside the repository; "" is the root
	Ref      string // explicit ref from "#ref" or a /tree/<branch> URL; "" lets ResolveRef decide
	Official bool   // from the Registry: pinned to the engine's minor version
}

// Display is the short form used in messages: the registry name, or
// owner/repo[/path], plus "#ref" when one was given.
func (s Spec) Display() string {
	var out string
	if s.Official {
		out = s.Input
		if i := strings.Index(out, "#"); i >= 0 {
			out = out[:i]
		}
	} else {
		out = s.Owner + "/" + s.Repo
		if s.Subpath != "" {
			out += "/" + s.Subpath
		}
	}
	if s.Ref != "" {
		out += "#" + s.Ref
	}
	return out
}

// CloneURL is the repository's HTTPS clone URL, offered when a download fails.
func (s Spec) CloneURL() string {
	return fmt.Sprintf("https://github.com/%s/%s.git", s.Owner, s.Repo)
}

// UnknownTemplateError is returned for a bare name that is not in the
// Registry. It is decided offline.
type UnknownTemplateError struct {
	Name string
}

func (e *UnknownTemplateError) Error() string {
	return fmt.Sprintf("unknown template %q; available templates: %s", e.Name, strings.Join(Names(), ", "))
}

// Parse accepts a registry name ("course"), "owner/repo[/path]", or a
// github.com URL (with an optional /tree/<branch>/<path>), each optionally
// followed by "#ref". The ref is cut off first because it may contain "/".
func Parse(input string) (Spec, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return Spec{}, errors.New("template name is empty")
	}
	base, ref, hasRef := strings.Cut(s, "#")
	if hasRef && (ref == "" || strings.ContainsAny(ref, " \t") || strings.HasPrefix(ref, "-")) {
		return Spec{}, fmt.Errorf("invalid template %q: the part after # must be a branch, tag, or commit", input)
	}
	spec := Spec{Input: input, Ref: ref}

	switch {
	case isGitHubHost(base):
		gh, err := download.ParseGitHubURL(base)
		if err != nil {
			return Spec{}, fmt.Errorf("invalid template %q: %v", input, err)
		}
		spec.Owner, spec.Repo, spec.Subpath = gh.Owner, gh.Repo, gh.Subpath
		if spec.Ref == "" && strings.Contains(base, "/tree/") {
			spec.Ref = gh.Branch
		}
	case strings.Contains(base, "/"):
		parts := strings.Split(base, "/")
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return Spec{}, invalidForm(input)
		}
		for _, p := range parts {
			if p == "" || p == "." || p == ".." || strings.ContainsAny(p, " \t\\") {
				return Spec{}, invalidForm(input)
			}
		}
		spec.Owner = parts[0]
		spec.Repo = strings.TrimSuffix(parts[1], ".git")
		if len(parts) > 2 {
			spec.Subpath = path.Join(parts[2:]...)
			if !fs.ValidPath(spec.Subpath) {
				return Spec{}, invalidForm(input)
			}
		}
	default:
		e, ok := lookup(base)
		if !ok {
			return Spec{}, &UnknownTemplateError{Name: base}
		}
		spec.Owner, spec.Repo, spec.Subpath, spec.Official = e.Owner, e.Repo, e.Subpath, true
	}
	return spec, nil
}

func invalidForm(input string) error {
	return fmt.Errorf("invalid template %q: use a template name (%s), owner/repo[/path][#ref], or a github.com URL",
		input, strings.Join(Names(), ", "))
}

func isGitHubHost(s string) bool {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "https://"), "http://")
	return strings.HasPrefix(s, "github.com/")
}
