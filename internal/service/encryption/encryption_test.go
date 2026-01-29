package encryption

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestEncryptionServiceCreation tests basic encryption service creation
func TestEncryptionServiceCreation(t *testing.T) {
	t.Run("valid 32-byte key", func(t *testing.T) {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}

		service, err := NewEncryptionService(key)
		if err != nil {
			t.Errorf("NewEncryptionService() error = %v, want nil", err)
		}
		if service == nil {
			t.Error("NewEncryptionService() returned nil service")
		}
	})

	t.Run("invalid key size - too short", func(t *testing.T) {
		key := make([]byte, 16) // Only 16 bytes
		_, err := NewEncryptionService(key)
		if err == nil {
			t.Error("NewEncryptionService() expected error for 16-byte key, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "32 bytes") {
			t.Errorf("NewEncryptionService() error = %v, want error mentioning 32 bytes", err)
		}
	})

	t.Run("invalid key size - too long", func(t *testing.T) {
		key := make([]byte, 64) // 64 bytes
		_, err := NewEncryptionService(key)
		if err == nil {
			t.Error("NewEncryptionService() expected error for 64-byte key, got nil")
		}
	})

	t.Run("empty key", func(t *testing.T) {
		key := make([]byte, 0)
		_, err := NewEncryptionService(key)
		if err == nil {
			t.Error("NewEncryptionService() expected error for empty key, got nil")
		}
	})
}

// TestEncryptDecryptBasic tests basic encryption and decryption
func TestEncryptDecryptBasic(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	service, err := NewEncryptionService(key)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple text", "hello world"},
		{"empty string", ""},
		{"special characters", "!@#$%^&*()_+-=[]{}|;:',.<>?/~`"},
		{"unicode", "Hello 世界 🌍"},
		{"long text", strings.Repeat("a", 10000)},
		{"credentials format", `{"username":"user123","password":"pass456","pin":"1234"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := service.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			if tt.plaintext != "" && encrypted == "" {
				t.Error("Encrypt() returned empty string for non-empty input")
			}

			decrypted, err := service.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("Round-trip failed: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

// TestEncryptionUniqueness tests that encryption produces different ciphertexts
func TestEncryptionUniqueness(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	service, err := NewEncryptionService(key)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	plaintext := "sensitive data"

	// Encrypt the same plaintext multiple times
	ciphertexts := make(map[string]bool)
	for i := 0; i < 100; i++ {
		encrypted, err := service.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}

		if ciphertexts[encrypted] {
			t.Errorf("Encrypt() produced duplicate ciphertext: %s", encrypted)
		}
		ciphertexts[encrypted] = true

		// Verify it still decrypts correctly
		decrypted, err := service.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Decrypt() error = %v", err)
		}
		if decrypted != plaintext {
			t.Errorf("Decrypt() = %q, want %q", decrypted, plaintext)
		}
	}
}

// TestDecryptInvalidData tests decryption with invalid inputs
func TestDecryptInvalidData(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	service, err := NewEncryptionService(key)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	tests := []struct {
		name       string
		ciphertext string
		wantErr    bool
	}{
		{"invalid base64", "not-valid-base64!!!", true},
		{"too short", "YWJj", true}, // "abc" in base64, too short for nonce
		{"random data", "cmFuZG9tZGF0YXRoYXRpc25vdGVuY3J5cHRlZA==", true},
		{"empty string", "", false}, // Empty should return empty
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Decrypt(tt.ciphertext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDifferentKeysCannotDecrypt tests that different keys cannot decrypt each other's data
func TestDifferentKeysCannotDecrypt(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	_, err := rand.Read(key1)
	if err != nil {
		t.Fatalf("failed to generate key1: %v", err)
	}
	_, err = rand.Read(key2)
	if err != nil {
		t.Fatalf("failed to generate key2: %v", err)
	}

	service1, err := NewEncryptionService(key1)
	if err != nil {
		t.Fatalf("failed to create service1: %v", err)
	}

	service2, err := NewEncryptionService(key2)
	if err != nil {
		t.Fatalf("failed to create service2: %v", err)
	}

	plaintext := "secret message"

	encrypted, err := service1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Try to decrypt with different key
	_, err = service2.Decrypt(encrypted)
	if err == nil {
		t.Error("Decrypt() with different key should fail, got nil error")
	}
}

// **Propriété 22: Round-trip chiffrement/déchiffrement**
// **Valide: Exigences 1.5**
//
// Property: For all valid plaintexts, encrypting and then decrypting
// must return the exact original plaintext with no data loss.
func TestProperty_RoundTripEncryptionDecryption(t *testing.T) {
	// Generate a random key for the test
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	service, err := NewEncryptionService(key)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("encrypt then decrypt returns original plaintext", prop.ForAll(
		func(plaintext string) bool {
			// Encrypt the plaintext
			encrypted, err := service.Encrypt(plaintext)
			if err != nil {
				t.Logf("Encrypt failed: %v", err)
				return false
			}

			// Decrypt the ciphertext
			decrypted, err := service.Decrypt(encrypted)
			if err != nil {
				t.Logf("Decrypt failed: %v", err)
				return false
			}

			// Verify round-trip: decrypted must equal original plaintext
			if decrypted != plaintext {
				t.Logf("Round-trip failed: original=%q, decrypted=%q", plaintext, decrypted)
				return false
			}

			return true
		},
		gen.AnyString(),
	))

	properties.Property("encrypted data is different from plaintext (except empty)", prop.ForAll(
		func(plaintext string) bool {
			if plaintext == "" {
				return true // Empty string is a special case
			}

			encrypted, err := service.Encrypt(plaintext)
			if err != nil {
				t.Logf("Encrypt failed: %v", err)
				return false
			}

			// Encrypted data should not equal plaintext
			if encrypted == plaintext {
				t.Logf("Encrypted data equals plaintext: %q", plaintext)
				return false
			}

			return true
		},
		gen.AnyString(),
	))

	properties.Property("same plaintext produces different ciphertexts", prop.ForAll(
		func(plaintext string) bool {
			if plaintext == "" {
				return true // Empty string is a special case
			}

			encrypted1, err := service.Encrypt(plaintext)
			if err != nil {
				t.Logf("First encrypt failed: %v", err)
				return false
			}

			encrypted2, err := service.Encrypt(plaintext)
			if err != nil {
				t.Logf("Second encrypt failed: %v", err)
				return false
			}

			// Due to random nonce, same plaintext should produce different ciphertexts
			if encrypted1 == encrypted2 {
				t.Logf("Same plaintext produced identical ciphertexts")
				return false
			}

			// But both should decrypt to the same plaintext
			decrypted1, err := service.Decrypt(encrypted1)
			if err != nil || decrypted1 != plaintext {
				t.Logf("First ciphertext failed to decrypt correctly")
				return false
			}

			decrypted2, err := service.Decrypt(encrypted2)
			if err != nil || decrypted2 != plaintext {
				t.Logf("Second ciphertext failed to decrypt correctly")
				return false
			}

			return true
		},
		gen.AnyString(),
	))

	properties.Property("encryption preserves data length information", prop.ForAll(
		func(plaintext string) bool {
			encrypted, err := service.Encrypt(plaintext)
			if err != nil {
				t.Logf("Encrypt failed: %v", err)
				return false
			}

			decrypted, err := service.Decrypt(encrypted)
			if err != nil {
				t.Logf("Decrypt failed: %v", err)
				return false
			}

			// Length must be preserved
			if len(decrypted) != len(plaintext) {
				t.Logf("Length not preserved: original=%d, decrypted=%d", len(plaintext), len(decrypted))
				return false
			}

			return true
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}
