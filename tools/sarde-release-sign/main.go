// Command sarde-release-sign is the vendor-side release signing tool. It is
// deliberately a separate program, never built into the sarde binary, so the
// private key and signing code paths do not ship to users.
//
// Generate a keypair (run once, keep the private key offline):
//
//	go run ./tools/sarde-release-sign -genkey -key release.key
//
// The public key is printed as a Go literal to paste into
// internal/updatesign/pubkey.go.
//
// Sign a release checksums file (GoReleaser runs this from its signs block):
//
//	go run ./tools/sarde-release-sign -in dist/checksums.txt -out dist/checksums.txt.sig
//
// The private key may also be supplied via the SARDE_RELEASE_SIGNING_KEY
// environment variable (base64) instead of -key. The tool never prints key
// material on any code path.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/getsarde/sarde/internal/updatesign"
)

func main() {
	var (
		genkey  = flag.Bool("genkey", false, "generate a new keypair and exit")
		keyPath = flag.String("key", "", "path to the private key file (base64)")
		in      = flag.String("in", "", "file to sign (e.g. dist/checksums.txt)")
		out     = flag.String("out", "", "signature output path (default <in>.sig)")
	)
	flag.Parse()

	if *genkey {
		runGenkey(*keyPath)
		return
	}

	if *in == "" {
		fatal("-in is required (or use -genkey)")
	}

	priv := loadPrivateKey(*keyPath)
	data, err := os.ReadFile(*in)
	if err != nil {
		fatal("reading %s: %v", *in, err)
	}
	sig := updatesign.Sign(data, priv)

	outPath := *out
	if outPath == "" {
		outPath = *in + ".sig"
	}
	if err := os.WriteFile(outPath, sig, 0o644); err != nil {
		fatal("writing %s: %v", outPath, err)
	}
	fmt.Printf("signature for %s written to %s\n", *in, outPath)
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
	fmt.Println("public key for internal/updatesign/pubkey.go:")
	fmt.Println("ed25519.PublicKey{")
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
	} else if env := os.Getenv("SARDE_RELEASE_SIGNING_KEY"); env != "" {
		encoded = env
	} else {
		fatal("no private key: pass -key <path> or set SARDE_RELEASE_SIGNING_KEY")
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
	fmt.Fprintf(os.Stderr, "sarde-release-sign: "+format+"\n", args...)
	os.Exit(1)
}
