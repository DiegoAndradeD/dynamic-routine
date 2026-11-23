// Package vault encrypts the state file so that it is safe to leave in a synced
// folder or a cloud drive: without the passphrase it is just noise.
//
// Format (all integers are single bytes):
//
//	"RTNV" | version | log2(N) | r | p | salt[16] | nonce[24] | ciphertext
//
// The key is derived once per session from the passphrase with scrypt and every
// save seals the data with XChaCha20-Poly1305 under a fresh random nonce. The
// header is bound to the ciphertext as additional data, so tampering with the
// KDF parameters or the salt is detected. Deriving once matters: scrypt is
// meant to be slow, and the app saves after every keystroke that changes data.
package vault

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/scrypt"
)

const (
	magic      = "RTNV"
	formatV1   = 1
	saltLen    = 16
	headerLen  = len(magic) + 4 + saltLen
	keyLen     = chacha20poly1305.KeySize
	defaultLog = 17 // N = 2^17, about 128 MiB and a third of a second
	maxLog     = 20 // N up to 2^20 (1 GiB with r=8): refuse anything larger from a hostile file
)

// ErrWrongPassphrase is returned when the data cannot be authenticated: the
// passphrase is wrong, or the file was corrupted or modified.
var ErrWrongPassphrase = errors.New("wrong passphrase, or the file is corrupted")

// IsEncrypted reports whether data looks like a vault file.
func IsEncrypted(data []byte) bool { return bytes.HasPrefix(data, []byte(magic)) }

// Cipher seals and opens vault files for one passphrase. It caches the derived
// key, so only the first use (and a file written with another salt) is slow.
type Cipher struct {
	pass   []byte
	header []byte
	key    []byte
}

// New returns a Cipher for the passphrase.
func New(passphrase []byte) *Cipher {
	return &Cipher{pass: append([]byte(nil), passphrase...)}
}

func (c *Cipher) derive(header []byte) error {
	logN, r, p := int(header[5]), int(header[6]), int(header[7])
	if logN < 10 || logN > maxLog || r < 1 || r > 8 || p < 1 || p > 4 {
		return errors.New("unsupported key derivation parameters")
	}
	key, err := scrypt.Key(c.pass, header[8:8+saltLen], 1<<logN, r, p, keyLen)
	if err != nil {
		return err
	}
	c.header, c.key = append([]byte(nil), header...), key
	return nil
}

// Seal encrypts plaintext. The first call generates a new random salt.
func (c *Cipher) Seal(plaintext []byte) ([]byte, error) {
	if c.key == nil {
		header := make([]byte, headerLen)
		copy(header, magic)
		header[4], header[5], header[6], header[7] = formatV1, defaultLog, 8, 1
		if _, err := rand.Read(header[8 : 8+saltLen]); err != nil {
			return nil, err
		}
		if err := c.derive(header); err != nil {
			return nil, err
		}
	}
	aead, err := chacha20poly1305.NewX(c.key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	out := make([]byte, 0, headerLen+len(nonce)+len(plaintext)+aead.Overhead())
	out = append(out, c.header...)
	out = append(out, nonce...)
	return aead.Seal(out, nonce, plaintext, c.header), nil
}

// Open decrypts data written by Seal.
func (c *Cipher) Open(data []byte) ([]byte, error) {
	if !IsEncrypted(data) {
		return nil, errors.New("not an encrypted routine file")
	}
	if len(data) < headerLen+chacha20poly1305.NonceSizeX+chacha20poly1305.Overhead {
		return nil, errors.New("encrypted file is truncated")
	}
	header := data[:headerLen]
	if header[4] != formatV1 {
		return nil, fmt.Errorf("unsupported file format version %d", header[4])
	}
	if c.key == nil || !bytes.Equal(c.header, header) {
		if err := c.derive(header); err != nil {
			return nil, err
		}
	}
	aead, err := chacha20poly1305.NewX(c.key)
	if err != nil {
		return nil, err
	}
	nonce := data[headerLen : headerLen+aead.NonceSize()]
	plain, err := aead.Open(nil, nonce, data[headerLen+aead.NonceSize():], header)
	if err != nil {
		return nil, ErrWrongPassphrase
	}
	return plain, nil
}
