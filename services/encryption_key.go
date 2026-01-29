package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

var (
	// ErrKeyNotSet is returned when the encryption key environment variable is not set
	ErrKeyNotSet = errors.New("ENCRYPTION_KEY environment variable is not set")
	// ErrInvalidKeyFormat is returned when the key is not valid hex
	ErrInvalidKeyFormat = errors.New("encryption key must be a valid 64-character hex string")
)

// LoadEncryptionKeyFromEnv loads the encryption key from the ENCRYPTION_KEY environment variable
// The key should be a 64-character hex string (32 bytes when decoded)
func LoadEncryptionKeyFromEnv() ([]byte, error) {
	keyHex := os.Getenv("ENCRYPTION_KEY")
	if keyHex == "" {
		return nil, ErrKeyNotSet
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKeyFormat, err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("%w: expected 64 hex characters (32 bytes), got %d bytes", ErrInvalidKeyFormat, len(key))
	}

	return key, nil
}

// GenerateEncryptionKey generates a new random 32-byte encryption key
// Returns the key as a hex-encoded string suitable for environment variables
func GenerateEncryptionKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	return hex.EncodeToString(key), nil
}
