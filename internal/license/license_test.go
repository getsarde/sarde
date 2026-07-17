package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testKeypair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func testLicense() *File {
	return &File{
		V:        1,
		Slug:     "slideviewer",
		Licensee: "test@example.com",
		Issued:   "2026-01-15",
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	pub, priv := testKeypair(t)
	f := testLicense()
	Sign(f, priv)
	if err := Verify(f, pub, "slideviewer", "1.2.0", time.Now()); err != nil {
		t.Fatalf("Verify after Sign: %v", err)
	}
}

func TestVerifyFailures(t *testing.T) {
	pub, priv := testKeypair(t)
	otherPub, _ := testKeypair(t)

	tests := []struct {
		name     string
		preSign  func(f *File) // applied before signing (covered by signature)
		postSign func(f *File) // applied after signing (tampering)
		pub      ed25519.PublicKey
		slug     string
		version  string
		now      time.Time
		wantErr  bool
	}{
		{
			name: "valid",
			pub:  pub, slug: "slideviewer", version: "1.2.0", now: time.Now(), wantErr: false,
		},
		{
			name:     "tampered licensee",
			postSign: func(f *File) { f.Licensee = "attacker@example.com" },
			pub:      pub, slug: "slideviewer", version: "1.2.0", now: time.Now(), wantErr: true,
		},
		{
			name: "wrong slug",
			pub:  pub, slug: "otherplugin", version: "1.0.0", now: time.Now(), wantErr: true,
		},
		{
			name: "wrong public key",
			pub:  otherPub, slug: "slideviewer", version: "1.2.0", now: time.Now(), wantErr: true,
		},
		{
			name:    "expired",
			preSign: func(f *File) { f.Expires = "2026-06-01" },
			pub:     pub, slug: "slideviewer", version: "1.2.0",
			now: time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC), wantErr: true,
		},
		{
			name:    "valid through expiry day",
			preSign: func(f *File) { f.Expires = "2026-06-01" },
			pub:     pub, slug: "slideviewer", version: "1.2.0",
			now: time.Date(2026, 6, 1, 23, 0, 0, 0, time.UTC), wantErr: false,
		},
		{
			name:    "version above max_version",
			preSign: func(f *File) { f.MaxVersion = "1.9.9" },
			pub:     pub, slug: "slideviewer", version: "2.0.0", now: time.Now(), wantErr: true,
		},
		{
			name:    "version within max_version",
			preSign: func(f *File) { f.MaxVersion = "1.9.9" },
			pub:     pub, slug: "slideviewer", version: "1.5.0", now: time.Now(), wantErr: false,
		},
		{
			name:     "malformed signature",
			postSign: func(f *File) { f.Sig = "!!not-base64!!" },
			pub:      pub, slug: "slideviewer", version: "1.2.0", now: time.Now(), wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := testLicense()
			if tt.preSign != nil {
				tt.preSign(f)
			}
			Sign(f, priv)
			if tt.postSign != nil {
				tt.postSign(f)
			}
			err := Verify(f, tt.pub, tt.slug, tt.version, tt.now)
			if tt.wantErr && err == nil {
				t.Fatal("expected verification error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected verification error: %v", err)
			}
		})
	}
}

func TestLoadMalformed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.license")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for malformed license JSON")
	}
}

func TestLocateProjectBeforeHome(t *testing.T) {
	project := t.TempDir()
	dir := filepath.Join(project, ".sarde", "licenses")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := testLicense()
	data, _ := json.Marshal(f)
	path := filepath.Join(dir, "slideviewer.license")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, found := Locate(project, "slideviewer")
	if !found || got != path {
		t.Errorf("Locate() = %q, %v; want %q, true", got, found, path)
	}

	if _, found := Locate(project, "unknown"); found {
		t.Error("Locate() found a license for an unknown slug")
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.2.0", "1.10.0", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.2", "1.2.0", 0},
		{"1.2.1", "1.2", 1},
	}
	for _, tt := range tests {
		if got := compareVersions(tt.a, tt.b); got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
