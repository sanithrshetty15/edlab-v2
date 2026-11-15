package gmailapi

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func testEncryptionKey() []byte {
	return bytes.Repeat([]byte{0x42}, 32)
}

func TestParseEncryptionKeyFormats(t *testing.T) {
	key := testEncryptionKey()
	values := []string{
		base64.StdEncoding.EncodeToString(key),
		base64.RawStdEncoding.EncodeToString(key),
		base64.URLEncoding.EncodeToString(key),
		base64.RawURLEncoding.EncodeToString(key),
		hex.EncodeToString(key),
		string(key),
	}
	for _, value := range values {
		decoded, err := parseEncryptionKey(value)
		if err != nil {
			t.Fatalf("parseEncryptionKey(%q): %v", value, err)
		}
		if !bytes.Equal(decoded, key) {
			t.Fatalf("parseEncryptionKey(%q) returned wrong key", value)
		}
	}
}

func TestParseEncryptionKeyRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"", "short", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 31))} {
		if _, err := parseEncryptionKey(value); err == nil {
			t.Fatalf("parseEncryptionKey(%q) unexpectedly succeeded", value)
		}
	}
}

func TestEncryptTokenRoundTrip(t *testing.T) {
	key := testEncryptionKey()
	plaintext := []byte("refresh-token-value")
	encrypted, err := encryptToken(key, plaintext)
	if err != nil {
		t.Fatalf("encryptToken: %v", err)
	}
	if bytes.Contains([]byte(encrypted), plaintext) {
		t.Fatal("encrypted token contains plaintext")
	}
	decrypted, err := decryptToken(key, encrypted)
	if err != nil {
		t.Fatalf("decryptToken: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decryptToken = %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptTokenRejectsWrongKeyAndTampering(t *testing.T) {
	key := testEncryptionKey()
	encrypted, err := encryptToken(key, []byte("refresh-token-value"))
	if err != nil {
		t.Fatalf("encryptToken: %v", err)
	}
	wrongKey := bytes.Repeat([]byte{0x24}, 32)
	if _, err := decryptToken(wrongKey, encrypted); err == nil {
		t.Fatal("decryptToken with wrong key unexpectedly succeeded")
	}
	raw, err := base64.RawStdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("decode encrypted token: %v", err)
	}
	raw[len(raw)-1] ^= 0xff
	tampered := base64.RawStdEncoding.EncodeToString(raw)
	if _, err := decryptToken(key, tampered); err == nil {
		t.Fatal("decryptToken with tampered ciphertext unexpectedly succeeded")
	}
}
