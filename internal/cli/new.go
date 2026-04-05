package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <collection> <title>",
	Short: "Create new content",
	Long:  "Create a new content file with frontmatter in the specified collection.",
	Args:  cobra.ExactArgs(2),
	RunE:  runNew,
}

func init() {
	rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	collection := args[0]
	title := args[1]

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	slug := slugify(title)
	if slug == "" {
		return fmt.Errorf("cannot generate slug from title %q", title)
	}

	relPath := filepath.Join("content", collection, slug+".md")
	absPath := filepath.Join(projectDir, relPath)

	// Check if file already exists.
	if _, err := os.Stat(absPath); err == nil {
		return fmt.Errorf("file already exists: %s", relPath)
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Write content file.
	now := time.Now().Format(time.RFC3339)
	content := fmt.Sprintf(`---
title: "%s"
date: "%s"
draft: true
---
`, title, now)

	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet {
		fmt.Printf("Created %s\n", relPath)
	}

	return nil
}

var nonAlnumHyphen = regexp.MustCompile(`[^a-z0-9-]+`)
var multiHyphen = regexp.MustCompile(`-{2,}`)

func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = nonAlnumHyphen.ReplaceAllString(s, "")
	s = multiHyphen.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
