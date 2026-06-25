package main

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

// generateTestKey creates a valid 64-character hex key (32 bytes) for testing
func generateTestKey() string {
	return strings.Repeat("a", 64)
}

func TestEncryptionDecryptionRoundtrip(t *testing.T) {
	key := generateTestKey()
	payload := []byte(`{"message": "secret AI prompt", "tokens": 100}`)

	// 1. Encrypt the payload
	nonceHex, ciphertextHex, err := EncryptPayload(payload, key)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if nonceHex == "" || ciphertextHex == "" {
		t.Fatal("Nonce or ciphertext is empty after encryption")
	}

	// 2. Decrypt the payload
	decrypted, err := DecryptPayload(ciphertextHex, nonceHex, key)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	// 3. Verify the result matches the original payload
	if !bytes.Equal(payload, decrypted) {
		t.Errorf("Expected decrypted payload to be %s, but got %s", payload, decrypted)
	}
}

func TestDecryptionWithWrongKey(t *testing.T) {
	correctKey := generateTestKey()
	wrongKey := strings.Repeat("b", 64) // Different 32-byte key
	payload := []byte("top secret data")

	nonceHex, ciphertextHex, _ := EncryptPayload(payload, correctKey)

	// Attempt to decrypt with the wrong key
	_, err := DecryptPayload(ciphertextHex, nonceHex, wrongKey)
	if err == nil {
		t.Error("Expected decryption to fail with wrong key, but it succeeded")
	}
}

func TestDecryptionWithManipulatedCiphertext(t *testing.T) {
	key := generateTestKey()
	payload := []byte("data to be manipulated")

	nonceHex, ciphertextHex, _ := EncryptPayload(payload, key)

	// Manipulate the ciphertext (change the first character)
	manipulatedCiphertext := "f" + ciphertextHex[1:]
	// Ensure it's still valid hex length, but data/tag is broken
	if len(manipulatedCiphertext) != len(ciphertextHex) {
		t.Fatal("Test setup error: manipulated hex string length mismatch")
	}

	// Attempt to decrypt
	_, err := DecryptPayload(manipulatedCiphertext, nonceHex, key)
	if err == nil {
		t.Error("Expected decryption to fail due to manipulated ciphertext (authentication tag check), but it succeeded")
	}
}

func TestInvalidHexKey(t *testing.T) {
	invalidKey := "this-is-not-a-valid-hex-string-because-it-has-invalid-chars!"
	payload := []byte("test")

	_, _, err := EncryptPayload(payload, invalidKey)
	if err == nil {
		t.Error("Expected encryption to fail with non-hex key, but it succeeded")
	}

	// Ensure the error originates from hex decoding
	if _, ok := err.(hex.InvalidByteError); !ok && err != hex.ErrLength {
		t.Errorf("Expected hex decoding error, got: %v", err)
	}
}