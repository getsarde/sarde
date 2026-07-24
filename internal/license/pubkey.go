package license

import "crypto/ed25519"

// PublicKey is the vendor ed25519 public key embedded in the sarde binary
// and used to verify premium plugin licenses.
//
// The matching private key is held offline by the vendor and is never present
// in this repository. Licenses are signed with the separate
// `tools/sarde-license-sign` program. Rotating this value invalidates every
// license already issued, so it must not change without a coordinated reissue.
var PublicKey = ed25519.PublicKey{
	0x90, 0x2d, 0x54, 0x75, 0x96, 0x42, 0x52, 0x2e,
	0x63, 0x7f, 0xc5, 0x62, 0xf1, 0x8d, 0x13, 0x74,
	0x55, 0xa2, 0x34, 0xa2, 0xf4, 0x01, 0x23, 0xc6,
	0x63, 0x2f, 0xed, 0x8d, 0xa2, 0x76, 0xae, 0x75,
}
