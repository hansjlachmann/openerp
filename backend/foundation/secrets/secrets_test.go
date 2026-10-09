package secrets

import (
	"errors"
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	defer SetKeyForTest("test key")()

	stored, err := Encrypt("abcd efgh ijkl mnop")
	if err != nil {
		t.Fatal(err)
	}
	if !IsEncrypted(stored) || strings.Contains(stored, "abcd") {
		t.Fatalf("stored = %q, want an encrypted value", stored)
	}
	if len(stored) != EncryptedLength(len("abcd efgh ijkl mnop")) {
		t.Errorf("length %d, EncryptedLength says %d", len(stored), EncryptedLength(19))
	}
	again, _ := Encrypt("abcd efgh ijkl mnop")
	if again == stored {
		t.Error("two encryptions of the same value are equal (nonce reused)")
	}
	if plain, err := Decrypt(stored); err != nil || plain != "abcd efgh ijkl mnop" {
		t.Errorf("Decrypt = %q, %v", plain, err)
	}
	if same, _ := Encrypt(stored); same != stored {
		t.Error("an encrypted value was encrypted again")
	}
	if empty, err := Encrypt(""); err != nil || empty != "" {
		t.Errorf("Encrypt(\"\") = %q, %v", empty, err)
	}
}

func TestDecryptPlainAndWrongKey(t *testing.T) {
	restore := SetKeyForTest("first key")
	stored, err := Encrypt("secret")
	restore()
	if err != nil {
		t.Fatal(err)
	}

	defer SetKeyForTest("second key")()
	if _, err := Decrypt(stored); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Decrypt with another key: %v, want ErrDecrypt", err)
	}
	// A value stored before encryption is read as it is
	if plain, err := Decrypt("old plain password"); err != nil || plain != "old plain password" {
		t.Errorf("Decrypt(plain) = %q, %v", plain, err)
	}
}

func TestNoKey(t *testing.T) {
	defer SetKeyForTest("")()
	if _, err := Encrypt("secret"); !errors.Is(err, ErrNoKey) {
		t.Errorf("Encrypt without key: %v, want ErrNoKey", err)
	}
	if _, err := Decrypt(prefix + "AAAA"); !errors.Is(err, ErrNoKey) {
		t.Errorf("Decrypt without key: %v, want ErrNoKey", err)
	}
}

// tools/tablegen computes the same length to size encrypted columns (maxEncryptedPlain)
func TestEncryptedLengthMatchesTablegen(t *testing.T) {
	if got := EncryptedLength(154); got != 250 {
		t.Errorf("EncryptedLength(154) = %d, want 250 (as tablegen's encryptedLength)", got)
	}
}
