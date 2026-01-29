package encryption_test

import (
	"fmt"
	"log"

	"valhafin/internal/service/encryption"
)

// Example demonstrates basic usage of the encryption service
func Example() {
	// Generate a new encryption key (do this once and store securely)
	keyHex, err := encryption.GenerateEncryptionKey()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Generated key (store in ENCRYPTION_KEY env var): %s\n", keyHex)

	// In production, load the key from environment variable
	// key, err := encryption.LoadEncryptionKeyFromEnv()

	// For this example, we'll decode the generated key
	// (In production, this would come from LoadEncryptionKeyFromEnv)
	// Output: Generated key (store in ENCRYPTION_KEY env var): [64 hex characters]
}

// ExampleEncryptionService_Encrypt demonstrates encrypting sensitive data
func ExampleEncryptionService_Encrypt() {
	// Create a 32-byte key (in production, use LoadEncryptionKeyFromEnv)
	key := make([]byte, 32)
	// In production: key, _ := encryption.LoadEncryptionKeyFromEnv()

	service, err := encryption.NewEncryptionService(key)
	if err != nil {
		log.Fatal(err)
	}

	// Encrypt sensitive credentials
	credentials := `{"username":"user123","password":"secret","pin":"1234"}`
	encrypted, err := service.Encrypt(credentials)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Encrypted: %t\n", len(encrypted) > 0)
	fmt.Println("Encrypted data is base64-encoded and safe to store in database")

	// Output:
	// Encrypted: true
	// Encrypted data is base64-encoded and safe to store in database
}

// ExampleEncryptionService_Decrypt demonstrates decrypting data
func ExampleEncryptionService_Decrypt() {
	// Create a 32-byte key (same key used for encryption)
	key := make([]byte, 32)

	service, err := encryption.NewEncryptionService(key)
	if err != nil {
		log.Fatal(err)
	}

	// First encrypt some data
	original := "sensitive information"
	encrypted, err := service.Encrypt(original)
	if err != nil {
		log.Fatal(err)
	}

	// Later, decrypt it
	decrypted, err := service.Decrypt(encrypted)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Original: %s\n", original)
	fmt.Printf("Decrypted: %s\n", decrypted)
	fmt.Printf("Match: %v\n", original == decrypted)

	// Output:
	// Original: sensitive information
	// Decrypted: sensitive information
	// Match: true
}
