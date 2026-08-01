// Package directive implements site- and theme-authored generic directives:
// data-driven ::: blocks defined by a directives/<name>.yaml schema, an
// <name>.html html/template, and an optional <name>.css sidecar. The package
// is goldmark-agnostic; the parsing/rendering side lives in
// internal/content/markdown/extensions/genericdirective.
package directive

import htmltemplate "html/template"

// TemplateData is the pipeline value every generic directive template
// executes against.
type TemplateData struct {
	Name  string             // directive name
	Label string             // bracket label or ""
	Attrs map[string]string  // attrutil.Parse of the opening fence; missing keys read ""
	Body  htmltemplate.HTML  // container: rendered children HTML; leaf: escaped raw text
}
