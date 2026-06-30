package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/syntax"
	"github.com/spf13/cobra"
)

var checkSyntaxCmd = &cobra.Command{
	Use:   "check-syntax [project-dir]",
	Short: "Check for unclosed or mismatched fenced block tags",
	Long:  "Scan markdown files for unclosed :::tag blocks, mismatched :::/tag closers, and orphaned closing tags.",
	RunE:  runCheckSyntax,
}

func init() {
	checkSyntaxCmd.Flags().String("content", "", "Override content directory path")
	checkSyntaxCmd.Flags().String("format", "pretty", "Output format: pretty, json")
	rootCmd.AddCommand(checkSyntaxCmd)
}

func runCheckSyntax(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")

	// Stdin mode: read markdown from stdin, write JSON to stdout (for sidecar/studio).
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		diags := syntax.Check("stdin", input, 0)
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"diagnostics": diags,
		})
	}

	// Standalone mode: scan content/ directory.
	projectDir := projectDirFromArgs(args)

	contentDir, _ := cmd.Flags().GetString("content")
	if contentDir == "" {
		contentDir = filepath.Join(projectDir, "content")
	} else if !filepath.IsAbs(contentDir) {
		contentDir, _ = filepath.Abs(contentDir)
	}

	var allDiags []syntax.Diagnostic
	err := filepath.Walk(contentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(projectDir, path)
		diags := syntax.Check(rel, data, 0)
		allDiags = append(allDiags, diags...)
		return nil
	})
	if err != nil {
		return fmt.Errorf("scanning content directory: %w", err)
	}

	if format == "json" {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"diagnostics": allDiags,
		})
	}

	if len(allDiags) == 0 {
		fmt.Fprintf(os.Stderr, "No syntax issues found.\n")
		return nil
	}

	for _, d := range allDiags {
		level := "WARN"
		if d.Level == "error" {
			level = "ERROR"
		}
		fmt.Fprintf(os.Stderr, "%s:%d [%s] %s\n", d.File, d.Line, level, d.Message)
	}

	hasErrors := false
	for _, d := range allDiags {
		if d.Level == "error" {
			hasErrors = true
			break
		}
	}
	if hasErrors {
		os.Exit(1)
	}

	return nil
}
