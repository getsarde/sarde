package asset

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Fingerprint computes a content hash for the given data.
// Returns the first 8 hex characters of the SHA-256 hash.
func Fingerprint(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:4])
}

// FingerprintFile reads a file and returns its content fingerprint.
func FingerprintFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return Fingerprint(data), nil
}

// FingerprintedName inserts a hash into a filename before the extension.
// Example: FingerprintedName("main.css", "a1b2c3d4") → "main.a1b2c3d4.css"
func FingerprintedName(name, hash string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return base + "." + hash + ext
}
