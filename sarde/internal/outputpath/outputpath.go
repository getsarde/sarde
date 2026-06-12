package outputpath

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/frostybee/sarde/internal/consts"
)

// SafeJoin resolves relPath under outputDir and rejects absolute paths,
// traversal, empty paths, and paths that escape the output directory.
func SafeJoin(outputDir, relPath string) (string, error) {
	if strings.TrimSpace(outputDir) == "" {
		return "", fmt.Errorf("empty output directory")
	}
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("empty output path")
	}

	raw := filepath.FromSlash(relPath)
	if filepath.VolumeName(raw) != "" {
		return "", fmt.Errorf("unsafe output path %q", relPath)
	}
	raw = strings.TrimLeft(raw, `/\`)
	rel := filepath.Clean(raw)
	if rel == "." || filepath.IsAbs(rel) || hasParentSegment(rel) {
		return "", fmt.Errorf("unsafe output path %q", relPath)
	}

	outRoot, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolving output directory: %w", err)
	}
	target, err := filepath.Abs(filepath.Join(outRoot, rel))
	if err != nil {
		return "", fmt.Errorf("resolving output path %q: %w", relPath, err)
	}
	if !IsWithin(outRoot, target) {
		return "", fmt.Errorf("output path escapes output directory: %q", relPath)
	}
	return target, nil
}

// IsWithin reports whether target is equal to or inside root.
func IsWithin(root, target string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	return isWithinAbs(absRoot, absTarget)
}

// isWithinAbs is like IsWithin but both root and target must already be
// absolute, Clean paths (e.g. results of filepath.Abs). Avoids redundant
// filepath.Abs calls in hot paths.
func isWithinAbs(absRoot, absTarget string) bool {
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

// SafeJoinWithRoot is like SafeJoin but accepts a pre-resolved absolute
// outputRoot, avoiding repeated filepath.Abs calls when the root is reused
// across many pages. outputRoot must be the result of filepath.Abs.
func SafeJoinWithRoot(outputRoot, relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("empty output path")
	}

	raw := filepath.FromSlash(relPath)
	if filepath.VolumeName(raw) != "" {
		return "", fmt.Errorf("unsafe output path %q", relPath)
	}
	raw = strings.TrimLeft(raw, `/\`)
	rel := filepath.Clean(raw)
	if rel == "." || filepath.IsAbs(rel) || hasParentSegment(rel) {
		return "", fmt.Errorf("unsafe output path %q", relPath)
	}

	target, err := filepath.Abs(filepath.Join(outputRoot, rel))
	if err != nil {
		return "", fmt.Errorf("resolving output path %q: %w", relPath, err)
	}
	if !isWithinAbs(outputRoot, target) {
		return "", fmt.Errorf("output path escapes output directory: %q", relPath)
	}
	return target, nil
}

// RemoveIfWithinAbs deletes a file only if it resolves inside the given
// absolute outputRoot. Avoids the filepath.Abs call on outputRoot that
// RemoveIfWithin performs.
func RemoveIfWithinAbs(outputRoot, path string) error {
	target, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}
	if !isWithinAbs(outputRoot, target) {
		return fmt.Errorf("output path escapes output directory: %s", path)
	}
	return os.Remove(target)
}

// ResolveOutputDir resolves a configured build output directory and rejects
// locations that could erase or overwrite source/project state.
func ResolveOutputDir(projectDir, configured string) (string, error) {
	if strings.TrimSpace(projectDir) == "" {
		return "", fmt.Errorf("empty project directory")
	}
	projectRoot, err := filepath.Abs(projectDir)
	if err != nil {
		return "", fmt.Errorf("resolving project directory: %w", err)
	}

	output := configured
	if output == "" {
		output = "dist"
	}
	if strings.TrimSpace(output) == "" {
		return "", fmt.Errorf("empty output directory")
	}

	// Reject any literal ".." segment in the raw (pre-Clean) path. This is
	// strictly stronger than checking the cleaned path — Clean never introduces
	// ".." but can erase one (e.g. "foo/../bar" → "bar"), and we want to reject
	// such paths too.
	rawOutput := filepath.FromSlash(output)
	if !filepath.IsAbs(rawOutput) && hasParentSegment(rawOutput) {
		return "", fmt.Errorf("output directory must not contain traversal: %q", configured)
	}
	cleaned := filepath.Clean(rawOutput)

	if !filepath.IsAbs(cleaned) {
		cleaned = filepath.Join(projectRoot, cleaned)
	}
	outputDir, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("resolving output directory: %w", err)
	}

	if samePath(outputDir, projectRoot) {
		return "", fmt.Errorf("output directory must not be the project root")
	}
	if IsWithin(outputDir, projectRoot) {
		return "", fmt.Errorf("output directory must not be a project ancestor")
	}

	for _, rel := range []string{".git", consts.DirContent, "layouts", "assets", "data", "static", "themes", "embedded"} {
		sourceDir := filepath.Join(projectRoot, rel)
		if samePath(outputDir, sourceDir) || IsWithin(sourceDir, outputDir) {
			return "", fmt.Errorf("output directory must not be inside source directory %q", rel)
		}
	}

	return outputDir, nil
}

// EnsureWithinOutputDir returns an absolute path only if path is inside outputDir.
func EnsureWithinOutputDir(outputDir, path string) (string, error) {
	outRoot, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolving output directory: %w", err)
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving output path: %w", err)
	}
	if !IsWithin(outRoot, target) {
		return "", fmt.Errorf("output path escapes output directory: %s", path)
	}
	return target, nil
}

// RemoveIfWithin deletes a file only if it resolves inside outputDir.
func RemoveIfWithin(outputDir, path string) error {
	target, err := EnsureWithinOutputDir(outputDir, path)
	if err != nil {
		return err
	}
	return os.Remove(target)
}

func hasParentSegment(path string) bool {
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if part == ".." {
			return true
		}
	}
	return false
}

func samePath(a, b string) bool {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false
	}
	ca, cb := filepath.Clean(absA), filepath.Clean(absB)
	// Case-insensitive comparison only where the filesystem convention is
	// case-insensitive; on Linux distinct casings are distinct paths.
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(ca, cb)
	}
	return ca == cb
}
