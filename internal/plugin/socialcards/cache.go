package socialcards

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// cardCacheVersion is folded into every card cache key. Bump it whenever
// renderCard's layout, constants, or drawing logic change in any visible way,
// or when an embedded asset (the Inter fonts, the Sarde mark and ribbon) is
// updated: keys only capture card inputs, so without a bump an older cache
// silently keeps serving art rendered by the previous code.
//
// History: 2 dropped the footer collection name when it repeats the title.
const cardCacheVersion = "2"

// cardCache stores encoded card bytes across builds, keyed by a content hash
// of every render input. Entries live under {projectDir}/.cache/social_cards
// as raw {key}.png / {key}.jpg files and are safe to delete at any time.
// There is no eviction, matching the asset cache.
type cardCache struct {
	dir string
}

func newCardCache(projectDir string) *cardCache {
	return &cardCache{dir: filepath.Join(projectDir, ".cache", "social_cards")}
}

func (c *cardCache) entryPath(key, format string) string {
	ext := ".png"
	if format == "jpeg" || format == "jpg" {
		ext = ".jpg"
	}
	return filepath.Join(c.dir, key+ext)
}

// Get returns the cached encoded bytes for key, or nil on a miss.
func (c *cardCache) Get(key, format string) []byte {
	data, err := os.ReadFile(c.entryPath(key, format))
	if err != nil {
		return nil
	}
	return data
}

// Put stores encoded card bytes best-effort: cache write failures never fail
// a build, the card is simply re-rendered next time.
func (c *cardCache) Put(key, format string, data []byte) {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(c.entryPath(key, format), data, 0o644)
}

// cardAssetHashes carries content digests of the binary render inputs that
// the text fields of CardParams cannot capture. An empty string is a valid
// value (absent image, embedded font slot); embedded assets need no digest
// because they only change with the binary, which cardCacheVersion covers.
type cardAssetHashes struct {
	Logo        string
	Watermark   string
	BgImage     string
	FontRegular string
	FontBold    string
}

// cardKey builds the content-addressed cache key for one card from every
// input that affects its pixels. The output path is deliberately excluded:
// two pages with identical inputs share one entry and still write their own
// output files. The date is keyed in its rendered footer form, so mtime
// churn on pages without an explicit date cannot invalidate their cards.
func cardKey(version string, p CardParams, format string, quality int, h cardAssetHashes) string {
	var b strings.Builder
	sep := func() { b.WriteByte(0) }
	str := func(s string) { b.WriteString(s); sep() }
	col := func(c color.NRGBA) { fmt.Fprintf(&b, "%02x%02x%02x%02x", c.R, c.G, c.B, c.A); sep() }

	str(version)
	str(p.Title)
	str(p.Description)
	str(p.SiteTitle)
	str(p.CollectionName)
	if p.DateExplicit && !p.Date.IsZero() {
		str(p.Date.Format("Jan 2, 2006"))
	} else {
		str("")
	}
	str(format)
	str(strconv.Itoa(quality))
	col(p.BgColor)
	col(p.AccentColor)
	if p.AccentColor2 != nil {
		col(*p.AccentColor2)
	} else {
		str("")
	}
	col(p.TextColor)
	str(strconv.FormatFloat(p.WatermarkOpacity, 'g', -1, 64))
	str(strconv.FormatFloat(p.BgImageOpacity, 'g', -1, 64))
	str(strconv.Itoa(len(p.GradientOverride)))
	for _, c := range p.GradientOverride {
		col(c)
	}
	str(h.Logo)
	str(h.Watermark)
	str(h.BgImage)
	str(h.FontRegular)
	str(h.FontBold)

	sum := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("%x", sum[:8])
}

// hashBytes returns a truncated sha256 hex digest of raw bytes, following the
// local truncated-hash convention used across packages.
func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:8])
}

// hashImage digests a decoded image's dimensions and pixels; nil hashes to
// "". Hashing pixels rather than source bytes means the digest also captures
// resize parameters such as logo_size and bg_image_fit for free.
func hashImage(img *image.NRGBA) string {
	if img == nil {
		return ""
	}
	h := sha256.New()
	fmt.Fprintf(h, "%dx%d:", img.Bounds().Dx(), img.Bounds().Dy())
	h.Write(img.Pix)
	return fmt.Sprintf("%x", h.Sum(nil)[:8])
}
