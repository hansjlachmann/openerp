// Package secrets encrypts secrets the application stores in the database (e.g. the SMTP
// password), so a database dump or backup does not reveal them.
//
// Values are encrypted with AES-256-GCM and stored as "enc:v1:<base64(nonce|ciphertext)>".
// The key comes from OPENERP_ENCRYPTION_KEY, else it is derived from JWT_SECRET (so an
// installation with a JWT_SECRET needs no new setting). Without either, values are stored
// as they are and a warning is logged at startup.
//
// Changing the key (or JWT_SECRET when it is the source) makes stored values unreadable:
// they then have to be entered again.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"sync"
)

// prefix marks an encrypted value (v1: AES-256-GCM, 12-byte nonce)
const prefix = "enc:v1:"

var (
	// ErrNoKey: neither OPENERP_ENCRYPTION_KEY nor JWT_SECRET is set
	ErrNoKey = errors.New("secrets: no encryption key (set OPENERP_ENCRYPTION_KEY or JWT_SECRET)")
	// ErrDecrypt: the value was encrypted with another key, or is damaged
	ErrDecrypt = errors.New("secrets: value cannot be decrypted (was the encryption key changed?)")
)

var (
	keyOnce   sync.Once
	keyBytes  []byte
	keySource string
)

// loadKey reads the key once: OPENERP_ENCRYPTION_KEY, else derived from JWT_SECRET.
// Any string is accepted; the AES key is its SHA-256 (with a label for JWT_SECRET, so
// the encryption key differs from the session signing key).
func loadKey() {
	keyOnce.Do(func() {
		if k := os.Getenv("OPENERP_ENCRYPTION_KEY"); k != "" {
			sum := sha256.Sum256([]byte(k))
			keyBytes, keySource = sum[:], "OPENERP_ENCRYPTION_KEY"
		} else if j := os.Getenv("JWT_SECRET"); j != "" {
			sum := sha256.Sum256([]byte("openerp secrets v1\x00" + j))
			keyBytes, keySource = sum[:], "JWT_SECRET"
		}
	})
}

// Source names where the key comes from ("OPENERP_ENCRYPTION_KEY", "JWT_SECRET"), or ""
// when there is none.
func Source() string {
	loadKey()
	return keySource
}

// SetKeyForTest replaces the key (nil: no key) and returns a function restoring it.
func SetKeyForTest(key string) func() {
	loadKey()
	oldKey, oldSource := keyBytes, keySource
	if key == "" {
		keyBytes, keySource = nil, ""
	} else {
		sum := sha256.Sum256([]byte(key))
		keyBytes, keySource = sum[:], "test"
	}
	return func() { keyBytes, keySource = oldKey, oldSource }
}

// IsEncrypted reports whether a stored value is encrypted
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, prefix)
}

// Encrypt returns the stored form of a secret. "" and already encrypted values are
// returned as they are; without a key it fails with ErrNoKey.
func Encrypt(plain string) (string, error) {
	if plain == "" || IsEncrypted(plain) {
		return plain, nil
	}
	gcm, err := newGCM()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return prefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

// Decrypt returns the secret of a stored value. A value that is not encrypted (stored
// before encryption, or without a key) is returned as it is.
func Decrypt(stored string) (string, error) {
	if !IsEncrypted(stored) {
		return stored, nil
	}
	gcm, err := newGCM()
	if err != nil {
		return "", err
	}
	sealed, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(stored, prefix))
	if err != nil || len(sealed) < gcm.NonceSize() {
		return "", ErrDecrypt
	}
	plain, err := gcm.Open(nil, sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():], nil)
	if err != nil {
		return "", ErrDecrypt
	}
	return string(plain), nil
}

// EncryptedLength is the stored length of a secret of n bytes
func EncryptedLength(n int) int {
	return len(prefix) + base64.RawStdEncoding.EncodedLen(12+n+16)
}

func newGCM() (cipher.AEAD, error) {
	loadKey()
	if keyBytes == nil {
		return nil, ErrNoKey
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
