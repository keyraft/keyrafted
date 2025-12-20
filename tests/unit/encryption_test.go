package unit

import (
	"keyrafted/internal/crypto"
	"testing"
)

func TestEncryptorEncryptDecrypt(t *testing.T) {
	masterKey := []byte("test-master-key-32-bytes-long!!!")
	encryptor, err := crypto.NewEncryptor(masterKey)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple text", "hello world"},
		{"empty string", ""},
		{"special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"unicode", "こんにちは世界 🌍"},
		{"long text", string(make([]byte, 10000))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Empty plaintext should return empty ciphertext
			if tt.plaintext == "" && ciphertext != "" {
				t.Errorf("Expected empty ciphertext for empty plaintext")
			}

			// Decrypt
			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Verify
			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptorDifferentCiphertext(t *testing.T) {
	masterKey := []byte("test-master-key-32-bytes-long!!!")
	encryptor, err := crypto.NewEncryptor(masterKey)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	plaintext := "test message"

	// Encrypt twice
	ciphertext1, _ := encryptor.Encrypt(plaintext)
	ciphertext2, _ := encryptor.Encrypt(plaintext)

	// Ciphertexts should be different due to random nonces
	if ciphertext1 == ciphertext2 {
		t.Error("Expected different ciphertexts for same plaintext")
	}

	// Both should decrypt to the same plaintext
	decrypted1, _ := encryptor.Decrypt(ciphertext1)
	decrypted2, _ := encryptor.Decrypt(ciphertext2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both ciphertexts should decrypt to original plaintext")
	}
}

func TestGenerateToken(t *testing.T) {
	token1, err := crypto.GenerateToken(32)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	token2, err := crypto.GenerateToken(32)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Tokens should be unique
	if token1 == token2 {
		t.Error("Expected unique tokens")
	}

	// Tokens should not be empty
	if len(token1) == 0 || len(token2) == 0 {
		t.Error("Expected non-empty tokens")
	}
}

func TestGenerateID(t *testing.T) {
	id1, err := crypto.GenerateID()
	if err != nil {
		t.Fatalf("GenerateID() error = %v", err)
	}

	id2, err := crypto.GenerateID()
	if err != nil {
		t.Fatalf("GenerateID() error = %v", err)
	}

	// IDs should be unique
	if id1 == id2 {
		t.Error("Expected unique IDs")
	}

	// IDs should be 32 characters (16 bytes in hex)
	if len(id1) != 32 || len(id2) != 32 {
		t.Errorf("Expected 32-character IDs, got %d and %d", len(id1), len(id2))
	}
}
