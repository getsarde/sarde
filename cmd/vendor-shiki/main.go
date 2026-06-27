package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	shikiVersion           = "3.7.0"
	vscodeTextmateVersion  = "10.0.2"
	onigurumaToESVersion   = "4.3.6"
	onigurumaParserVersion = "0.12.2"
	regexVersion           = "6.1.0"
	regexRecursionVersion  = "6.0.2"
	regexUtilitiesVersion  = "2.3.0"
)

type keepRule struct {
	Dir  string
	Exts []string
}

type vendorPkg struct {
	Name    string
	Version string
	DestDir string
	Keep    []keepRule
}

var packages = []vendorPkg{
	// --- Shiki core packages ---
	{Name: "@shikijs/core", Version: shikiVersion, DestDir: "@shikijs/core",
		Keep: []keepRule{{Dir: "dist", Exts: []string{".mjs"}}}},
	{Name: "@shikijs/engine-javascript", Version: shikiVersion, DestDir: "@shikijs/engine-javascript",
		Keep: []keepRule{{Dir: "dist", Exts: []string{".mjs"}}}},
	{Name: "@shikijs/langs", Version: shikiVersion, DestDir: "@shikijs/langs",
		Keep: []keepRule{{Dir: "dist", Exts: []string{".mjs"}}}},
	{Name: "@shikijs/themes", Version: shikiVersion, DestDir: "@shikijs/themes",
		Keep: []keepRule{{Dir: "dist", Exts: []string{".mjs"}}}},
	{Name: "@shikijs/types", Version: shikiVersion, DestDir: "@shikijs/types",
		Keep: []keepRule{{Dir: "dist", Exts: []string{".mjs"}}}},
	{Name: "@shikijs/vscode-textmate", Version: vscodeTextmateVersion, DestDir: "@shikijs/vscode-textmate",
		Keep: []keepRule{{Dir: "dist", Exts: []string{".js", ".mjs"}}}},

	// --- oniguruma-to-es dependency chain ---
	{Name: "oniguruma-to-es", Version: onigurumaToESVersion, DestDir: "oniguruma-to-es",
		Keep: []keepRule{{Dir: "dist", Exts: []string{".js"}}}},
	{Name: "oniguruma-parser", Version: onigurumaParserVersion, DestDir: "oniguruma-parser",
		Keep: []keepRule{{Dir: "dist", Exts: []string{".js"}}}},
	{Name: "regex", Version: regexVersion, DestDir: "regex",
		Keep: []keepRule{
			{Dir: "dist", Exts: []string{".js"}},
			{Dir: "src", Exts: []string{".js"}},
		}},
	{Name: "regex-recursion", Version: regexRecursionVersion, DestDir: "regex-recursion",
		Keep: []keepRule{{Dir: "src", Exts: []string{".js"}}}},
	{Name: "regex-utilities", Version: regexUtilitiesVersion, DestDir: "regex-utilities",
		Keep: []keepRule{{Dir: "src", Exts: []string{".js"}}}},
}

func main() {
	vendorRoot := filepath.Join("embedded", "vendor", "shiki")

	fmt.Println("Vendoring Shiki ESM packages...")
	fmt.Printf("  Target: %s\n\n", vendorRoot)

	if err := os.RemoveAll(vendorRoot); err != nil {
		fatalf("removing existing vendor dir: %v", err)
	}
	if err := os.MkdirAll(vendorRoot, 0o755); err != nil {
		fatalf("creating vendor dir: %v", err)
	}

	var totalFiles int
	var totalBytes int64

	for _, pkg := range packages {
		url := tarballURL(pkg.Name, pkg.Version)
		destDir := filepath.Join(vendorRoot, pkg.DestDir)

		fmt.Printf("  Downloading %s@%s ...\n", pkg.Name, pkg.Version)

		tmpFile, err := downloadToTemp(url)
		if err != nil {
			fatalf("downloading %s: %v", pkg.Name, err)
		}

		files, bytes, err := extractSelective(tmpFile, destDir, pkg.Keep)
		os.Remove(tmpFile)
		if err != nil {
			fatalf("extracting %s: %v", pkg.Name, err)
		}

		totalFiles += files
		totalBytes += bytes
		fmt.Printf("    OK %s@%s -> %s/ (%d files, %s)\n", pkg.Name, pkg.Version, pkg.DestDir, files, formatBytes(bytes))
	}

	stubFiles, stubBytes := writeStubs(vendorRoot)
	totalFiles += stubFiles
	totalBytes += stubBytes

	writeVersionFile(vendorRoot)
	totalFiles++

	fmt.Printf("\nDone. %d packages, %d files, %s total.\n", len(packages), totalFiles, formatBytes(totalBytes))
}

func tarballURL(name, version string) string {
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		scope := parts[0]
		pkg := parts[1]
		return fmt.Sprintf("https://registry.npmjs.org/%s/%s/-/%s-%s.tgz", scope, pkg, pkg, version)
	}
	return fmt.Sprintf("https://registry.npmjs.org/%s/-/%s-%s.tgz", name, name, version)
}

func downloadToTemp(url string) (string, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp("", "shiki-vendor-*.tgz")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	tmp.Close()
	return tmp.Name(), nil
}

func extractSelective(tarPath, destDir string, rules []keepRule) (int, int64, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return 0, 0, fmt.Errorf("opening %s: %w", tarPath, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, 0, fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var fileCount int
	var byteCount int64

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fileCount, byteCount, fmt.Errorf("reading tar: %w", err)
		}

		relPath := stripPackagePrefix(filepath.ToSlash(hdr.Name))
		if relPath == "" {
			continue
		}

		if hdr.Typeflag == tar.TypeDir {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		if !shouldKeep(relPath, rules) {
			continue
		}

		dest := filepath.Join(destDir, filepath.FromSlash(relPath))
		if !isInsideDir(destDir, dest) {
			return fileCount, byteCount, fmt.Errorf("tar entry %q escapes destination", hdr.Name)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fileCount, byteCount, fmt.Errorf("creating dir for %s: %w", relPath, err)
		}

		out, err := os.Create(dest)
		if err != nil {
			return fileCount, byteCount, fmt.Errorf("creating %s: %w", dest, err)
		}
		n, err := io.Copy(out, tr)
		out.Close()
		if err != nil {
			return fileCount, byteCount, fmt.Errorf("writing %s: %w", dest, err)
		}

		fileCount++
		byteCount += n
	}

	return fileCount, byteCount, nil
}

func stripPackagePrefix(name string) string {
	idx := strings.Index(name, "/")
	if idx < 0 {
		return ""
	}
	return name[idx+1:]
}

func shouldKeep(relPath string, rules []keepRule) bool {
	if relPath == "package.json" {
		return true
	}
	for _, rule := range rules {
		prefix := rule.Dir + "/"
		if !strings.HasPrefix(relPath, prefix) {
			continue
		}
		for _, ext := range rule.Exts {
			if strings.HasSuffix(relPath, ext) {
				return true
			}
		}
	}
	return false
}

func writeStubs(vendorRoot string) (int, int64) {
	stubs := map[string]string{
		filepath.Join("hast-util-to-html", "package.json"): `{"name":"hast-util-to-html","type":"module","exports":{".":"./index.mjs"}}`,
		filepath.Join("hast-util-to-html", "index.mjs"): `export function toHtml() {
  throw new Error("hast-util-to-html is not vendored. Use codeToTokens() instead.");
}
`,
		filepath.Join("@types", "hast", "package.json"): `{"name":"@types/hast","type":"module","exports":{".":"./index.mjs"}}`,
		filepath.Join("@types", "hast", "index.mjs"): `export default {};
`,
	}

	var files int
	var bytes int64
	for relPath, content := range stubs {
		dest := filepath.Join(vendorRoot, relPath)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			fatalf("creating stub dir: %v", err)
		}
		if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			fatalf("writing stub %s: %v", relPath, err)
		}
		files++
		bytes += int64(len(content))
		fmt.Printf("    OK stub -> %s\n", relPath)
	}
	return files, bytes
}

func writeVersionFile(vendorRoot string) {
	var b strings.Builder
	b.WriteString("# Vendored Shiki ESM packages for Sarde\n")
	b.WriteString("# Generated by: go run ./cmd/vendor-shiki\n")
	b.WriteString("# Re-run this tool to update.\n\n")

	for _, pkg := range packages {
		fmt.Fprintf(&b, "%s=%s\n", pkg.Name, pkg.Version)
	}

	dest := filepath.Join(vendorRoot, "VERSION")
	if err := os.WriteFile(dest, []byte(b.String()), 0o644); err != nil {
		fatalf("writing VERSION: %v", err)
	}
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func isInsideDir(base, target string) bool {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
