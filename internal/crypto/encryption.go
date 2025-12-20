package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

// Encryptor handles encryption and decryption of secrets
type Encryptor struct {
	masterKey []byte
}

// NewEncryptor creates a new encryptor with a master key
func NewEncryptor(masterKey []byte) (*Encryptor, error) {
	if len(masterKey) < 16 {
		return nil, fmt.Errorf("master key must be at least 16 bytes")
	}

	// Derive a 32-byte key from the master key
	derivedKey := pbkdf2.Key(masterKey, []byte("keyraft-salt"), 10000, 32, sha256.New)

	return &Encryptor{
		masterKey: derivedKey,
	}, nil
}

// NewEncryptorFromEnv creates an encryptor from environment variable or file
func NewEncryptorFromEnv(envVar, filePath string) (*Encryptor, error) {
	var masterKey []byte

	// Try environment variable first
	if envVar != "" {
		keyStr := os.Getenv(envVar)
		if keyStr != "" {
			masterKey = []byte(keyStr)
		}
	}

	// Try file if env var not set
	if len(masterKey) == 0 && filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read master key file: %w", err)
		}
		masterKey = data
	}

	// Generate random key if none provided (for development only)
	if len(masterKey) == 0 {
		masterKey = make([]byte, 32)
		if _, err := rand.Read(masterKey); err != nil {
			return nil, fmt.Errorf("failed to generate random master key: %w", err)
		}
		fmt.Fprintf(os.Stderr, "WARNING: Using auto-generated master key. Set %s or use --master-key-file for production.\n", envVar)
	}

	return NewEncryptor(masterKey)
}

// Encrypt encrypts plaintext using AES-256-GCM
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(e.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// GenerateToken generates a secure random token
func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateID generates a unique ID
func GenerateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}
