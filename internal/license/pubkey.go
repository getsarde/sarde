package license

import "crypto/ed25519"

// PublicKey is the vendor ed25519 public key embedded in the sarde binary
// and used to verify premium plugin licenses.
//
// PLACEHOLDER: this value is not a real vendor key. Before selling licenses,
// generate a keypair with `go run ./tools/sarde-license-sign -genkey`, keep
// the private key offline, and replace this value with the printed public
// key. All verifications fail against the placeholder.
var PublicKey = ed25519.PublicKey{
	0x9d, 0x2f, 0x41, 0x7a, 0xc3, 0x58, 0xe6, 0x01,
	0xb4, 0x77, 0x2a, 0x93, 0x5e, 0xd0, 0x18, 0xcf,
	0x64, 0x0b, 0xaa, 0x36, 0x81, 0xf5, 0x4c, 0xd9,
	0x2e, 0x63, 0x90, 0x07, 0xbb, 0x14, 0xe8, 0x52,
}
