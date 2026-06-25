package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

// EncryptPayload encrypts the provided data using AES-GCM 
// and returns the generated nonce and ciphertext as hex-encoded strings.
func EncryptPayload(payload []byte, keyHex string) (string, string, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	// Generate a 12-byte cryptographically secure random nonce
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}

	// Seal encrypts the payload and automatically appends the authentication tag to the ciphertext
	ciphertext := aesgcm.Seal(nil, nonce, payload, nil)

	return hex.EncodeToString(nonce), hex.EncodeToString(ciphertext), nil
}

// DecryptPayload takes hex-encoded strings for ciphertext and nonce, 
// decrypts them using AES-GCM, and returns the original plaintext byte slice.
func DecryptPayload(ciphertextHex, nonceHex, keyHex string) ([]byte, error) {
	key, _ := hex.DecodeString(keyHex)
	nonce, _ := hex.DecodeString(nonceHex)
	ciphertext, _ := hex.DecodeString(ciphertextHex)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Open decrypts the ciphertext and simultaneously verifies the integrity of the authentication tag
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: payload was manipulated or the key is invalid")
	}

	return plaintext, nil
}