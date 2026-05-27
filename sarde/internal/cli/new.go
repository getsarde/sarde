package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/project"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <collection> <title>",
	Short: "Create new content",
	Long:  "Create a new content file with frontmatter in the specified collection.\n\nSubcommands `course` and `lesson` scaffold course directories and auto-numbered lessons.",
	Args:  cobra.ExactArgs(2),
	RunE:  runNew,
}

var newCourseCmd = &cobra.Command{
	Use:   "course <name>",
	Short: "Scaffold a new course",
	Long:  "Create a new course directory under content/courses/<name>/ with config.yaml and _index.md.",
	Args:  cobra.ExactArgs(1),
	RunE:  runNewCourse,
}

var newLessonCmd = &cobra.Command{
	Use:   "lesson <course> <name>",
	Short: "Scaffold a new lesson inside a course (auto-numbered)",
	Long:  "Create a new lesson markdown file inside content/courses/<course>/. Filename is automatically prefixed with the next two-digit number (01-, 02-, ...).",
	Args:  cobra.ExactArgs(2),
	RunE:  runNewLesson,
}

func init() {
	newCmd.AddCommand(newCourseCmd)
	newCmd.AddCommand(newLessonCmd)
	rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	collection := args[0]
	title := args[1]

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	slug := content.Slugify(title)
	if slug == "" {
		return fmt.Errorf("cannot generate slug from title %q", title)
	}

	relPath := filepath.Join(consts.DirContent, collection, slug+".md")
	absPath := filepath.Join(projectDir, relPath)

	// Check if file already exists.
	if _, err := os.Stat(absPath); err == nil {
		return fmt.Errorf("file already exists: %s", relPath)
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Write content file using archetype/schema-aware scaffolding.
	if err := project.ScaffoldFile(projectDir, collection, title, absPath); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet {
		fmt.Printf("Created %s\n", relPath)
	}

	return nil
}

// titleCaseSlug converts a slug like "my-new-course" to "My New Course".
func titleCaseSlug(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func runNewCourse(cmd *cobra.Command, args []string) error {
	name := args[0]
	slug := content.Slugify(name)
	if slug == "" {
		return fmt.Errorf("cannot generate slug from name %q", name)
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	courseDir := filepath.Join(projectDir, consts.DirContent, consts.CollectionCourses, slug)
	if _, err := os.Stat(courseDir); err == nil {
		return fmt.Errorf("course already exists: %s", filepath.Join(consts.DirContent, consts.CollectionCourses, slug))
	}

	if err := os.MkdirAll(courseDir, 0o755); err != nil {
		return fmt.Errorf("creating course directory: %w", err)
	}

	title := titleCaseSlug(slug)

	configPath := filepath.Join(courseDir, consts.FileCollConfig)
	configBody := fmt.Sprintf("title: %s\ndescription: \"\"\nicon: book\n", title)
	if err := os.WriteFile(configPath, []byte(configBody), 0o644); err != nil {
		return fmt.Errorf("writing config.yaml: %w", err)
	}

	indexPath := filepath.Join(courseDir, "_index.md")
	if err := project.ScaffoldFile(projectDir, consts.CollectionCourses, title, indexPath); err != nil {
		return fmt.Errorf("writing _index.md: %w", err)
	}

	if quiet, _ := cmd.Flags().GetBool("quiet"); !quiet {
		fmt.Printf("Created course %s\n", filepath.Join(consts.DirContent, consts.CollectionCourses, slug))
	}
	return nil
}

func runNewLesson(cmd *cobra.Command, args []string) error {
	courseSlug := content.Slugify(args[0])
	lessonName := args[1]
	lessonSlug := content.Slugify(lessonName)
	if courseSlug == "" {
		return fmt.Errorf("invalid course slug %q", args[0])
	}
	if lessonSlug == "" {
		return fmt.Errorf("cannot generate slug from lesson name %q", lessonName)
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	courseDir := filepath.Join(projectDir, consts.DirContent, consts.CollectionCourses, courseSlug)
	info, err := os.Stat(courseDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("course %q does not exist (run 'sarde new course %s' first)", courseSlug, courseSlug)
	}

	maxNum := 0
	entries, err := os.ReadDir(courseDir)
	if err != nil {
		return fmt.Errorf("reading course directory: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".md") || n == "_index.md" || n == "index.md" {
			continue
		}
		base := strings.TrimSuffix(n, ".md")
		if w, _, ok := content.ExtractNumericPrefix(base); ok && w > maxNum {
			maxNum = w
		}
	}
	next := maxNum + 1

	filename := fmt.Sprintf("%02d-%s.md", next, lessonSlug)
	lessonPath := filepath.Join(courseDir, filename)
	if _, err := os.Stat(lessonPath); err == nil {
		return fmt.Errorf("lesson already exists: %s", filepath.Join(consts.DirContent, consts.CollectionCourses, courseSlug, filename))
	}

	title := titleCaseSlug(lessonSlug)
	if err := project.ScaffoldFile(projectDir, consts.CollectionCourses, title, lessonPath); err != nil {
		return fmt.Errorf("writing lesson: %w", err)
	}

	if quiet, _ := cmd.Flags().GetBool("quiet"); !quiet {
		fmt.Printf("Created lesson %s\n", filepath.Join(consts.DirContent, consts.CollectionCourses, courseSlug, filename))
	}
	return nil
}
