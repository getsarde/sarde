package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/directive"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/spf13/cobra"
)

var newDirectiveCmd = &cobra.Command{
	Use:   "directive <name>",
	Short: "Scaffold a new generic directive",
	Long:  "Create directives/<name>.yaml, <name>.html, and <name>.css starter files for a site-authored ::: directive. Use it in any page as :::<name> ... ::: once created.",
	Args:  cobra.ExactArgs(1),
	RunE:  runNewDirective,
}

func runNewDirective(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !directive.ValidName(name) {
		return fmt.Errorf("invalid directive name %q: use lowercase letters, digits, and hyphens, starting with a letter (e.g. pullquote, my-box)", name)
	}

	if cat, err := engine.LoadDirectiveCatalog(); err == nil {
		for _, c := range cat.Categories {
			for _, d := range c.Directives {
				if d.Name == name {
					return fmt.Errorf("%q is a built-in directive (%s); pick another name, built-ins always win name conflicts", name, c.Label)
				}
			}
		}
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	dirDir := filepath.Join(projectDir, consts.DirDirectives)
	yamlPath := filepath.Join(dirDir, name+".yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return fmt.Errorf("directive already exists: %s", filepath.Join(consts.DirDirectives, name+".yaml"))
	}

	if err := os.MkdirAll(dirDir, 0o755); err != nil {
		return fmt.Errorf("creating directives directory: %w", err)
	}

	label := titleCaseSlug(name)
	files := map[string]string{
		name + ".yaml": fmt.Sprintf(directiveYAMLTemplate, name, label),
		name + ".html": fmt.Sprintf(directiveHTMLTemplate, name, name, name, name),
		name + ".css":  fmt.Sprintf(directiveCSSTemplate, name, name, name),
	}
	for fileName, body := range files {
		if err := os.WriteFile(filepath.Join(dirDir, fileName), []byte(body), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", fileName, err)
		}
	}

	if quiet, _ := cmd.Flags().GetBool("quiet"); !quiet {
		rel := func(ext string) string { return filepath.Join(consts.DirDirectives, name+ext) }
		fmt.Printf("Created directive %q:\n", name)
		fmt.Printf("  %s   (schema: fields, label, kind)\n", rel(".yaml"))
		fmt.Printf("  %s   (html/template rendering the directive)\n", rel(".html"))
		fmt.Printf("  %s    (styles, bundled into the site CSS automatically)\n", rel(".css"))
		fmt.Println()
		fmt.Printf("Use it in any page:\n")
		fmt.Printf("  :::%s[Optional label] example=\"value\"\n", name)
		fmt.Printf("  Body content here.\n")
		fmt.Printf("  :::\n")
		fmt.Println()
		fmt.Println("  'sarde dev' reloads directives automatically.")
		fmt.Println("  Docs: /docs/extensions/custom-directives")
	}
	return nil
}

// %s slots: name, label.
const directiveYAMLTemplate = `# Generic directive definition. The name must match this filename.
name: %s
kind: container   # "container": body is markdown; "leaf": body is raw text
label: "%s"
description: "Describe what this directive renders."
# category: custom

# Optional [bracket] label support: :::name[My Label]
# bracket:
#   label: Title
#   required: false

# Fields become key="value" attrs on the opening fence.
fields:
  - { name: example, label: Example, type: string }
  # - { name: variant, label: Variant, type: enum, options: [plain, fancy], default: plain }
`

// %s slots: name (css class), name, name, name.
const directiveHTMLTemplate = `<div class="%s">
  {{- if .Label }}
  <div class="%s-label">{{ .Label }}</div>
  {{- end }}
  {{- if .Attrs.example }}
  <div class="%s-example">{{ .Attrs.example }}</div>
  {{- end }}
  <div class="%s-body">{{ .Body }}</div>
</div>
`

// %s slots: name, name, name. Uses --sd-* theme tokens so the directive
// follows the active theme and dark mode automatically.
const directiveCSSTemplate = `/* Styles for the %s directive.
   Use --sd-* theme tokens (e.g. var(--sd-accent), var(--sd-gray-4)) so your
   directive adapts to theme presets and dark mode automatically. See the
   theme tokens reference in the docs: /docs/guides/themes-and-styling */
.%s {
  border-inline-start: 3px solid var(--sd-accent);
  padding: 0.75rem 1rem;
  margin-block: 1rem;
}

.%s-label {
  font-weight: 600;
  color: var(--sd-accent);
}
`
