package updatesign

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
)

func testKeys(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating keypair: %v", err)
	}
	return pub, priv
}

// withTrustedKeys swaps the embedded key set for the test's own keys and
// restores it afterward.
func withTrustedKeys(t *testing.T, keys ...ed25519.PublicKey) {
	t.Helper()
	orig := PublicKeys
	PublicKeys = keys
	t.Cleanup(func() { PublicKeys = orig })
}

func TestSignVerifyRoundTrip(t *testing.T) {
	pub, priv := testKeys(t)
	withTrustedKeys(t, pub)

	data := []byte("abc123  sarde_linux_amd64.tar.gz\ndef456  sarde_windows_amd64.zip\n")
	sig := Sign(data, priv)
	if err := Verify(data, sig); err != nil {
		t.Fatalf("Verify() after Sign() failed: %v", err)
	}
}

func TestVerifyTamperedData(t *testing.T) {
	pub, priv := testKeys(t)
	withTrustedKeys(t, pub)

	data := []byte("original checksums")
	sig := Sign(data, priv)
	tampered := []byte("tampered checksums")
	if err := Verify(tampered, sig); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("Verify() on tampered data = %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyWrongKey(t *testing.T) {
	pub, _ := testKeys(t)
	_, otherPriv := testKeys(t)
	withTrustedKeys(t, pub)

	data := []byte("checksums")
	sig := Sign(data, otherPriv)
	if err := Verify(data, sig); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("Verify() with wrong key = %v, want ErrInvalidSignature", err)
	}
}

func TestVerifySecondTrustedKey(t *testing.T) {
	oldPub, _ := testKeys(t)
	newPub, newPriv := testKeys(t)
	withTrustedKeys(t, oldPub, newPub)

	data := []byte("checksums")
	sig := Sign(data, newPriv)
	if err := Verify(data, sig); err != nil {
		t.Fatalf("Verify() against second trusted key failed: %v", err)
	}
}

func TestVerifyMalformedSignature(t *testing.T) {
	pub, _ := testKeys(t)
	withTrustedKeys(t, pub)

	if err := Verify([]byte("data"), []byte("short")); err == nil {
		t.Fatal("Verify() with malformed signature succeeded, want error")
	}
}

func TestEmbeddedKeyIsValid(t *testing.T) {
	if len(PublicKeys) == 0 {
		t.Fatal("no embedded release public key")
	}
	for i, pub := range PublicKeys {
		if len(pub) != ed25519.PublicKeySize {
			t.Errorf("PublicKeys[%d] has length %d, want %d", i, len(pub), ed25519.PublicKeySize)
		}
	}
}
