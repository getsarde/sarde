// Command sarde-license-sign is the vendor-side license signing tool. It is
// deliberately a separate program, never built into the sarde binary, so the
// private key and signing code paths do not ship to users.
//
// Generate a keypair (run once, keep the private key offline):
//
//	go run ./tools/sarde-license-sign -genkey -key vendor.key
//
// The public key is printed as a Go literal to paste into
// internal/license/pubkey.go.
//
// Sign a license:
//
//	go run ./tools/sarde-license-sign -key vendor.key -slug slideviewer \
//	    -licensee "jane@example.com" -expires 2027-12-31 -out slideviewer.license
//
// The private key may also be supplied via the SARDE_SIGNING_KEY environment
// variable (base64) instead of -key.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/getsarde/sarde/internal/license"
)

func main() {
	var (
		genkey     = flag.Bool("genkey", false, "generate a new keypair and exit")
		keyPath    = flag.String("key", "", "path to the private key file (base64)")
		slug       = flag.String("slug", "", "plugin slug the license is for")
		licensee   = flag.String("licensee", "", "licensee name or email")
		issued     = flag.String("issued", time.Now().Format("2006-01-02"), "issue date (YYYY-MM-DD)")
		expires    = flag.String("expires", "", "expiry date (YYYY-MM-DD), empty for perpetual")
		maxVersion = flag.String("max-version", "", "highest plugin version covered, empty for unlimited")
		seats      = flag.Int("seats", 0, "seat count, 0 for unspecified")
		out        = flag.String("out", "", "output license file path (default {slug}.license)")
	)
	flag.Parse()

	if *genkey {
		runGenkey(*keyPath)
		return
	}

	if *slug == "" || *licensee == "" {
		fatal("both -slug and -licensee are required (or use -genkey)")
	}

	priv := loadPrivateKey(*keyPath)
	f := &license.File{
		V:          1,
		Slug:       *slug,
		Licensee:   *licensee,
		Issued:     *issued,
		Expires:    *expires,
		MaxVersion: *maxVersion,
		Seats:      *seats,
	}
	license.Sign(f, priv)

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		fatal("encoding license: %v", err)
	}
	outPath := *out
	if outPath == "" {
		outPath = *slug + ".license"
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		fatal("writing %s: %v", outPath, err)
	}
	fmt.Printf("signed license for %q written to %s\n", *slug, outPath)
}

func runGenkey(keyPath string) {
	if keyPath == "" {
		fatal("-genkey requires -key <path> for the private key output file")
	}
	if _, err := os.Stat(keyPath); err == nil {
		fatal("refusing to overwrite existing key file %s", keyPath)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fatal("generating keypair: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(priv)
	if err := os.WriteFile(keyPath, []byte(encoded+"\n"), 0o600); err != nil {
		fatal("writing private key: %v", err)
	}
	fmt.Printf("private key written to %s (keep offline, never commit)\n\n", keyPath)
	fmt.Println("public key for internal/license/pubkey.go:")
	fmt.Println("var PublicKey = ed25519.PublicKey{")
	for i := 0; i < len(pub); i += 8 {
		fmt.Print("\t")
		for j := i; j < i+8 && j < len(pub); j++ {
			fmt.Printf("0x%02x, ", pub[j])
		}
		fmt.Println()
	}
	fmt.Println("}")
}

func loadPrivateKey(keyPath string) ed25519.PrivateKey {
	var encoded string
	if keyPath != "" {
		data, err := os.ReadFile(keyPath)
		if err != nil {
			fatal("reading key file: %v", err)
		}
		encoded = string(data)
	} else if env := os.Getenv("SARDE_SIGNING_KEY"); env != "" {
		encoded = env
	} else {
		fatal("no private key: pass -key <path> or set SARDE_SIGNING_KEY")
	}
	raw, err := base64.StdEncoding.DecodeString(trimSpace(encoded))
	if err != nil {
		fatal("decoding private key: %v", err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		fatal("private key has wrong length %d, want %d", len(raw), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(raw)
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "sarde-license-sign: "+format+"\n", args...)
	os.Exit(1)
}
