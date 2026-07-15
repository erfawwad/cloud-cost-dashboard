// Package crypto encrypts provider credentials (cloud API keys/secrets)
// before they are stored in the database, using AES-256-GCM with a key
// derived from the app's ENCRYPTION_KEY config value.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

type Box struct {
	gcm cipher.AEAD
}

func New(key string) (*Box, error) {
	// Derive a 32-byte key regardless of the raw length/format of the
	// configured secret, so operators can use any passphrase for ENCRYPTION_KEY.
	sum := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{gcm: gcm}, nil
}

func (b *Box) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, b.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := b.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (b *Box) Decrypt(encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	nonceSize := b.gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return b.gcm.Open(nil, nonce, ciphertext, nil)
}
