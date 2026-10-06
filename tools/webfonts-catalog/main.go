// Command webfonts-catalog regenerates internal/webfonts/families.tsv, the
// list of Google Fonts families that theme.web_fonts may load. It is run by
// hand when the catalog needs a refresh, and its output is committed:
//
//	go run ./tools/webfonts-catalog
//
// The source is the metadata behind fonts.google.com (no API key needed).
// Each output line is `Family<TAB>category`, sorted by family, where the
// category is one of sans-serif, serif, display, handwriting, monospace.
// Bunny Fonts mirrors the same library, so one list serves both providers.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const metadataURL = "https://fonts.google.com/metadata/fonts"

type metadata struct {
	FamilyMetadataList []struct {
		Family   string `json:"family"`
		Category string `json:"category"`
	} `json:"familyMetadataList"`
}

func main() {
	out := flag.String("out", "internal/webfonts/families.tsv", "output file")
	flag.Parse()

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(metadataURL)
	if err != nil {
		fail("fetch: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fail("fetch: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fail("read: %v", err)
	}
	// The endpoint has at times prefixed its JSON with an XSSI guard.
	text := strings.TrimPrefix(string(body), ")]}'")

	var meta metadata
	if err := json.Unmarshal([]byte(text), &meta); err != nil {
		fail("decode: %v", err)
	}

	lines := make([]string, 0, len(meta.FamilyMetadataList))
	seen := make(map[string]bool)
	for _, f := range meta.FamilyMetadataList {
		name := strings.TrimSpace(f.Family)
		if name == "" || seen[strings.ToLower(name)] {
			continue
		}
		seen[strings.ToLower(name)] = true
		lines = append(lines, name+"\t"+category(f.Category))
	}
	if len(lines) < 1000 {
		fail("only %d families in the metadata; refusing to write a short catalog", len(lines))
	}
	sort.Slice(lines, func(i, j int) bool { return strings.ToLower(lines[i]) < strings.ToLower(lines[j]) })

	if err := os.WriteFile(*out, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		fail("write: %v", err)
	}
	fmt.Printf("wrote %d families to %s\n", len(lines), *out)
}

// category maps Google's display names ("Sans Serif") to the lowercase
// CSS-style names the engine uses.
func category(c string) string {
	switch strings.ToLower(strings.TrimSpace(c)) {
	case "sans serif":
		return "sans-serif"
	case "serif":
		return "serif"
	case "display":
		return "display"
	case "handwriting":
		return "handwriting"
	case "monospace":
		return "monospace"
	default:
		return "sans-serif"
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "webfonts-catalog: "+format+"\n", args...)
	os.Exit(1)
}
