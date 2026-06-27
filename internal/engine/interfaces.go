package engine

// MarkdownRenderer converts Markdown content to HTML via the Goldmark pipeline.
// Used as a parameter type by shortcode/processor to decouple from content/markdown.
type MarkdownRenderer interface {
	Render(markdown string) (RenderResult, error)
}
