package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/frostybee/sarde/internal/content/markdown"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/spf13/cobra"
)

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render markdown to HTML",
	Long:  "Read markdown from stdin and output rendered HTML with headings as JSON.",
	RunE:  runRender,
}

func init() {
	rootCmd.AddCommand(renderCmd)
}

type renderResult struct {
	HTML     string           `json:"html"`
	Headings []engine.Heading `json:"headings"`
}

func runRender(cmd *cobra.Command, args []string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	r := markdown.NewRenderer()
	mdResult, err := r.Render(string(input))
	if err != nil {
		return fmt.Errorf("rendering markdown: %w", err)
	}

	result := renderResult{
		HTML:     mdResult.HTML,
		Headings: mdResult.Headings,
	}

	return json.NewEncoder(os.Stdout).Encode(result)
}
