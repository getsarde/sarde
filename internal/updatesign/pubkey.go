package updatesign

import "crypto/ed25519"

// PublicKeys is the set of trusted vendor ed25519 public keys embedded in the
// sarde binary and used to verify release signatures during self-update.
//
// The matching private key is held offline by the vendor and is never present
// in this repository. Release checksums are signed with the separate
// `tools/sarde-release-sign` program. During a key rotation, ship a release
// that lists both the old and the new key here; binaries older than that
// release can only verify releases signed with a key they already trust.
var PublicKeys = []ed25519.PublicKey{
	{
		0x3b, 0x0a, 0x04, 0x6c, 0x05, 0xdb, 0x1c, 0x6d,
		0xfe, 0x45, 0x7e, 0x60, 0x7f, 0x40, 0xf5, 0x39,
		0xda, 0x33, 0x86, 0xed, 0xf5, 0x45, 0x82, 0xda,
		0x7e, 0xe5, 0x62, 0x21, 0x85, 0xca, 0x0e, 0xb4,
	},
}
