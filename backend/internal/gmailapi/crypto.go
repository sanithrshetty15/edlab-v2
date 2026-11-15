package gmailapi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

var tokenAAD = []byte("edwinslab:gmail-refresh-token:v1")

func parseEncryptionKey(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("GMAIL_TOKEN_ENCRYPTION_KEY must be set")
	}
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		if decoded, err := encoding.DecodeString(value); err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}
	if len(value) == 64 {
		if decoded, err := hex.DecodeString(value); err == nil {
			return decoded, nil
		}
	}
	if len(value) == 32 {
		return []byte(value), nil
	}
	return nil, fmt.Errorf("GMAIL_TOKEN_ENCRYPTION_KEY must decode to exactly 32 bytes")
}

func encryptToken(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, plaintext, tokenAAD)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func decryptToken(key []byte, encoded string) ([]byte, error) {
	data, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, fmt.Errorf("decode encrypted token: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < aead.NonceSize() {
		return nil, fmt.Errorf("encrypted token is incomplete")
	}
	nonce, ciphertext := data[:aead.NonceSize()], data[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, tokenAAD)
	if err != nil {
		return nil, fmt.Errorf("decrypt token: %w", err)
	}
	return plaintext, nil
}
