package utils

import (
	"testing"
)

const testKey = "01234567890123456789012345678901" // 32 bytes

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	tests := []string{
		"sk_test_abc123xyz",
		"sk_live_very_secret_key_12345",
		"",
		"short",
		"a-very-long-api-key-that-contains-special-chars!@#$%^&*()",
	}

	for _, plaintext := range tests {
		encrypted, err := Encrypt(plaintext, testKey)
		if err != nil {
			t.Fatalf("Encrypt(%q) failed: %v", plaintext, err)
		}

		if encrypted == plaintext && plaintext != "" {
			t.Errorf("Encrypt should produce different output from input")
		}

		decrypted, err := Decrypt(encrypted, testKey)
		if err != nil {
			t.Fatalf("Decrypt failed: %v", err)
		}

		if decrypted != plaintext {
			t.Errorf("Decrypt(%q) = %q, want %q", encrypted, decrypted, plaintext)
		}
	}
}

func TestEncrypt_DifferentOutputEachTime(t *testing.T) {
	plaintext := "same-input"
	enc1, _ := Encrypt(plaintext, testKey)
	enc2, _ := Encrypt(plaintext, testKey)

	if enc1 == enc2 {
		t.Error("Same plaintext should produce different ciphertexts (random nonce)")
	}
}

func TestEncrypt_WrongKeyLength(t *testing.T) {
	_, err := Encrypt("test", "short-key")
	if err == nil {
		t.Error("expected error for wrong key length")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	encrypted, _ := Encrypt("secret", testKey)
	wrongKey := "99999999999999999999999999999999"
	_, err := Decrypt(encrypted, wrongKey)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	_, err := Decrypt("not-valid-base64!!!", testKey)
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}
