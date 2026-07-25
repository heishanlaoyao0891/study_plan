package aikey

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

func Encrypt(plaintext, secret string) (string, error) {
	if plaintext == "" {
		return "", fmt.Errorf("api key is required")
	}
	gcm, err := newGCM(secret)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(encoded, secret string) (string, error) {
	gcm, err := newGCM(secret)
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("stored API key is invalid")
	}
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt stored API key: %w", err)
	}
	return string(plaintext), nil
}

func newGCM(secret string) (cipher.AEAD, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("AI_KEY_ENCRYPTION_SECRET is not configured")
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
