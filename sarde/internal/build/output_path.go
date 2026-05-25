package build

import (
	"github.com/frostybee/sarde/internal/outputpath"
)

// ResolveOutputDir resolves and validates the configured build output directory.
func ResolveOutputDir(projectDir, configured string) (string, error) {
	return outputpath.ResolveOutputDir(projectDir, configured)
}

func safeOutputPath(outputDir, relPath string) (string, error) {
	return outputpath.SafeJoin(outputDir, relPath)
}

func writeOutputFile(outputDir, relPath string, data []byte) (string, error) {
	path, err := safeOutputPath(outputDir, relPath)
	if err != nil {
		return "", err
	}
	return path, writeFile(path, data)
}
