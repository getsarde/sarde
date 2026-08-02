// Package updatesign verifies ed25519 signatures on release artifacts for
// the self-update flow. The release checksums file is signed offline with the
// separate tools/sarde-release-sign program; the sarde binary only ever
// verifies. This is deliberately a distinct trust domain from
// internal/license: compromising or rotating one key must not affect the
// other.
package updatesign

import (
	"crypto/ed25519"
	"errors"
	"fmt"
)

// ErrInvalidSignature is returned when a signature matches none of the
// trusted release keys.
var ErrInvalidSignature = errors.New("release signature does not match any trusted key")

// Sign returns the ed25519 signature of data. Only the offline release
// signing tool calls this; it is never reached from the shipped binary.
func Sign(data []byte, priv ed25519.PrivateKey) []byte {
	return ed25519.Sign(priv, data)
}

// Verify checks sig over data against the trusted release public keys.
// Accepting a set rather than a single key means a future key rotation only
// needs a release that ships both keys, not a special transition scheme.
func Verify(data, sig []byte) error {
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("release signature has length %d, want %d", len(sig), ed25519.SignatureSize)
	}
	for _, pub := range PublicKeys {
		if ed25519.Verify(pub, data, sig) {
			return nil
		}
	}
	return ErrInvalidSignature
}
