package cryptobox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

type Box struct{ aead cipher.AEAD }

func New(encoded string) (*Box, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return nil, errors.New("encryption key must be base64-encoded 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead}, nil
}

func (b *Box) Seal(plaintext []byte, purpose string) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return b.aead.Seal(nonce, nonce, plaintext, []byte(purpose)), nil
}

func (b *Box) Open(payload []byte, purpose string) ([]byte, error) {
	n := b.aead.NonceSize()
	if len(payload) < n+b.aead.Overhead() {
		return nil, errors.New("encrypted payload is truncated")
	}
	return b.aead.Open(nil, payload[:n], payload[n:], []byte(purpose))
}
