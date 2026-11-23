package vault

import (
	"bytes"
	"errors"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	c := New([]byte("correct horse"))
	sealed, err := c.Seal([]byte("secret tasks"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("secret")) || !IsEncrypted(sealed) {
		t.Fatal("output must be opaque and recognisable")
	}
	// a fresh Cipher (another process, another machine) opens it too
	plain, err := New([]byte("correct horse")).Open(sealed)
	if err != nil || string(plain) != "secret tasks" {
		t.Fatalf("open = %q, %v", plain, err)
	}
}

func TestSealsAreUnique(t *testing.T) {
	c := New([]byte("pw"))
	a, _ := c.Seal([]byte("same"))
	b, _ := c.Seal([]byte("same"))
	if bytes.Equal(a, b) {
		t.Fatal("two seals of the same data must differ (random nonce)")
	}
	if !bytes.Equal(a[:headerLen], b[:headerLen]) {
		t.Fatal("the key (salt) is reused within a session")
	}
}

func TestWrongPassphraseAndTampering(t *testing.T) {
	sealed, _ := New([]byte("right")).Seal([]byte("data"))

	if _, err := New([]byte("wrong")).Open(sealed); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("wrong passphrase: %v", err)
	}

	for _, i := range []int{6, 10, len(sealed) - 1} { // KDF params, salt, ciphertext
		bad := append([]byte(nil), sealed...)
		bad[i] ^= 1
		if _, err := New([]byte("right")).Open(bad); err == nil {
			t.Fatalf("flipping byte %d went unnoticed", i)
		}
	}
	if _, err := New([]byte("right")).Open(sealed[:20]); err == nil {
		t.Fatal("truncated file must fail")
	}
}

func TestHostileKDFParametersRejected(t *testing.T) {
	sealed, _ := New([]byte("pw")).Seal([]byte("x"))
	for name, patch := range map[string]func([]byte){
		"huge N": func(b []byte) { b[5] = 40 },
		"huge r": func(b []byte) { b[6] = 255 },
		"huge p": func(b []byte) { b[7] = 255 },
	} {
		bad := append([]byte(nil), sealed...)
		patch(bad)
		if _, err := New([]byte("pw")).Open(bad); err == nil {
			t.Errorf("%s: must refuse absurd scrypt parameters", name)
		}
	}
}
