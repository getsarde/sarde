// Package sitetemplate resolves and fetches the site templates behind
// `sarde new site --template`. Official templates live in the
// getsarde/sarde-templates repository, one folder per template; any public
// GitHub repository folder works through the owner/repo[/path][#ref] form.
// Templates are downloaded as GitHub archives, so git is not required.
package sitetemplate

import "sort"

// Entry is an official template: a name users type, and the repository
// folder it maps to.
type Entry struct {
	Name    string
	Owner   string
	Repo    string
	Subpath string
}

// Registry lists the official templates. Adding one is a row here plus a
// folder in getsarde/sarde-templates.
var Registry = []Entry{
	{Name: "course", Owner: "getsarde", Repo: "sarde-templates", Subpath: "course"},
}

// Names returns the official template names, sorted, for help text and
// error messages.
func Names() []string {
	names := make([]string, 0, len(Registry))
	for _, e := range Registry {
		names = append(names, e.Name)
	}
	sort.Strings(names)
	return names
}

func lookup(name string) (Entry, bool) {
	for _, e := range Registry {
		if e.Name == name {
			return e, true
		}
	}
	return Entry{}, false
}
