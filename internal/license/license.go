// Package license implements offline verification of ed25519-signed license
// files for premium external plugins. The sarde binary embeds only the
// vendor's public key; signing happens in the separate, never-shipped
// tools/sarde-license-sign program.
package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// File is a signed plugin license as stored on disk (JSON).
type File struct {
	V          int    `json:"v"`
	Slug       string `json:"slug"`
	Licensee   string `json:"licensee"`
	Issued     string `json:"issued"`                // YYYY-MM-DD
	Expires    string `json:"expires,omitempty"`     // empty = never
	MaxVersion string `json:"max_version,omitempty"` // empty = unlimited
	Seats      int    `json:"seats,omitempty"`
	Sig        string `json:"sig"` // base64 ed25519 signature over canonical()
}

// canonical returns the fixed-order byte string covered by the signature.
// A pipe-joined string is used instead of raw JSON so the signer and the
// verifier never depend on JSON key ordering.
func (f *File) canonical() []byte {
	return fmt.Appendf(nil, "%d|%s|%s|%s|%s|%s|%d",
		f.V, f.Slug, f.Licensee, f.Issued, f.Expires, f.MaxVersion, f.Seats)
}

// Sign fills Sig with the base64 ed25519 signature of the payload. Used by
// the vendor signing tool and tests; the sarde binary only ever verifies.
func Sign(f *File, priv ed25519.PrivateKey) {
	f.Sig = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, f.canonical()))
}

// Verify checks the signature and validity constraints of a license for the
// given plugin slug and version. now is a parameter for testability.
func Verify(f *File, pub ed25519.PublicKey, slug, version string, now time.Time) error {
	if f.Slug != slug {
		return fmt.Errorf("license is for plugin %q, not %q", f.Slug, slug)
	}
	sig, err := base64.StdEncoding.DecodeString(f.Sig)
	if err != nil {
		return fmt.Errorf("malformed license signature")
	}
	if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, f.canonical(), sig) {
		return fmt.Errorf("invalid license signature")
	}
	if f.Expires != "" {
		exp, err := time.Parse("2006-01-02", f.Expires)
		if err != nil {
			return fmt.Errorf("malformed license expiry date %q", f.Expires)
		}
		// The license stays valid through the expiry day itself.
		if now.After(exp.AddDate(0, 0, 1)) {
			return fmt.Errorf("license expired on %s", f.Expires)
		}
	}
	if f.MaxVersion != "" && version != "" && compareVersions(version, f.MaxVersion) > 0 {
		return fmt.Errorf("license covers plugin versions up to %s, installed version is %s", f.MaxVersion, version)
	}
	return nil
}

// Load reads and parses a license file from disk.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing license file %s: %w", filepath.Base(path), err)
	}
	return &f, nil
}

// CandidatePaths returns the locations checked for a plugin's license, in
// priority order: project-level .sarde/licenses/ first (CI, per-project
// overrides), then the user home directory (one purchase, every project).
func CandidatePaths(projectDir, slug string) []string {
	paths := []string{
		filepath.Join(projectDir, ".sarde", "licenses", slug+".license"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".sarde", "licenses", slug+".license"))
	}
	return paths
}

// Locate returns the first existing license file for slug.
func Locate(projectDir, slug string) (string, bool) {
	for _, p := range CandidatePaths(projectDir, slug) {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, true
		}
	}
	return "", false
}

// VerifyFor locates, loads, and verifies the license for a premium plugin.
// The returned error is user-facing and names the exact problem.
func VerifyFor(projectDir, slug, version string) error {
	path, found := Locate(projectDir, slug)
	if !found {
		return fmt.Errorf("no license file found (looked for %s)",
			strings.Join(CandidatePaths(projectDir, slug), " and "))
	}
	f, err := Load(path)
	if err != nil {
		return err
	}
	return Verify(f, PublicKey, slug, version, time.Now())
}

// compareVersions compares two dotted numeric versions (1.2.0 style),
// returning -1, 0, or 1. Non-numeric segments compare as strings; missing
// segments count as zero.
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		av, bv := "0", "0"
		if i < len(as) && as[i] != "" {
			av = as[i]
		}
		if i < len(bs) && bs[i] != "" {
			bv = bs[i]
		}
		an, aerr := strconv.Atoi(av)
		bn, berr := strconv.Atoi(bv)
		switch {
		case aerr == nil && berr == nil:
			if an != bn {
				if an < bn {
					return -1
				}
				return 1
			}
		default:
			if av != bv {
				if av < bv {
					return -1
				}
				return 1
			}
		}
	}
	return 0
}
