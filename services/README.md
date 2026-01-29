# Services Package

This package contains core services for the Valhafin application.

## Encryption Service

The encryption service provides AES-256-GCM encryption and decryption for sensitive data such as account credentials and API keys.

### Features

- **AES-256-GCM encryption**: Industry-standard authenticated encryption
- **Secure key management**: 32-byte (256-bit) keys loaded from environment variables
- **Random nonces**: Each encryption uses a unique random nonce for security
- **Base64 encoding**: Encrypted data is base64-encoded for safe storage in databases
- **Authentication**: GCM mode provides both confidentiality and authenticity

### Usage

#### 1. Generate an Encryption Key

```bash
# Generate a new 32-byte key (do this once)
go run -c 'package main; import "valhafin/services"; func main() { key, _ := services.GenerateEncryptionKey(); println(key) }'
```

Or use the helper function in your code:

```go
keyHex, err := services.GenerateEncryptionKey()
if err != nil {
    log.Fatal(err)
}
fmt.Println("Store this in ENCRYPTION_KEY environment variable:", keyHex)
```

#### 2. Set Environment Variable

```bash
export ENCRYPTION_KEY="your_64_character_hex_key_here"
```

Or add to `.env` file:

```
ENCRYPTION_KEY=your_64_character_hex_key_here
```

#### 3. Use the Encryption Service

```go
package main

import (
    "log"
    "valhafin/services"
)

func main() {
    // Load key from environment
    key, err := services.LoadEncryptionKeyFromEnv()
    if err != nil {
        log.Fatal(err)
    }

    // Create encryption service
    encryptionService, err := services.NewEncryptionService(key)
    if err != nil {
        log.Fatal(err)
    }

    // Encrypt sensitive data
    credentials := `{"username":"user123","password":"secret","pin":"1234"}`
    encrypted, err := encryptionService.Encrypt(credentials)
    if err != nil {
        log.Fatal(err)
    }

    // Store encrypted in database
    // ...

    // Later, decrypt when needed
    decrypted, err := encryptionService.Decrypt(encrypted)
    if err != nil {
        log.Fatal(err)
    }

    // Use decrypted credentials
    // ...
}
```

### Security Considerations

1. **Key Storage**: Never commit the encryption key to version control. Always use environment variables or secure key management systems.

2. **Key Rotation**: If you need to rotate keys:
   - Generate a new key
   - Decrypt all data with the old key
   - Re-encrypt with the new key
   - Update the environment variable

3. **Key Length**: The service requires exactly 32 bytes (256 bits) for AES-256. The hex-encoded key will be 64 characters.

4. **Nonce Uniqueness**: Each encryption automatically generates a unique random nonce, ensuring the same plaintext produces different ciphertexts.

5. **Authentication**: GCM mode provides authenticated encryption, protecting against tampering.

### Testing

The encryption service includes comprehensive tests:

- **Unit tests**: Basic functionality, edge cases, error handling
- **Property-based tests**: Validates correctness properties across random inputs
  - Round-trip encryption/decryption
  - Ciphertext uniqueness
  - Data length preservation
  - Plaintext vs ciphertext difference

Run tests:

```bash
go test -v ./services/...
```

### API Reference

#### `NewEncryptionService(key []byte) (*EncryptionService, error)`

Creates a new encryption service with the provided 32-byte key.

**Parameters:**
- `key`: 32-byte encryption key

**Returns:**
- `*EncryptionService`: The encryption service instance
- `error`: Error if key is invalid

#### `Encrypt(plaintext string) (string, error)`

Encrypts plaintext using AES-256-GCM.

**Parameters:**
- `plaintext`: The data to encrypt

**Returns:**
- `string`: Base64-encoded ciphertext (nonce + encrypted data + auth tag)
- `error`: Error if encryption fails

#### `Decrypt(ciphertext string) (string, error)`

Decrypts base64-encoded ciphertext.

**Parameters:**
- `ciphertext`: Base64-encoded encrypted data

**Returns:**
- `string`: The original plaintext
- `error`: Error if decryption fails or authentication fails

#### `LoadEncryptionKeyFromEnv() ([]byte, error)`

Loads the encryption key from the `ENCRYPTION_KEY` environment variable.

**Returns:**
- `[]byte`: The 32-byte encryption key
- `error`: Error if key is not set or invalid

#### `GenerateEncryptionKey() (string, error)`

Generates a new random 32-byte encryption key.

**Returns:**
- `string`: Hex-encoded key (64 characters)
- `error`: Error if random generation fails

### Error Types

- `ErrInvalidKeySize`: Key is not 32 bytes
- `ErrInvalidCiphertext`: Ciphertext is malformed or too short
- `ErrDecryptionFailed`: Decryption or authentication failed
- `ErrKeyNotSet`: ENCRYPTION_KEY environment variable not set
- `ErrInvalidKeyFormat`: Key is not valid hex format

### Property 22: Round-trip Encryption/Decryption

**Validates: Requirements 1.5**

The encryption service satisfies the following correctness property:

> For all valid plaintexts, encrypting and then decrypting must return the exact original plaintext with no data loss.

This property is verified through property-based testing with 100+ random test cases, ensuring:
- Perfect round-trip: `decrypt(encrypt(x)) = x`
- Ciphertext uniqueness: Same plaintext produces different ciphertexts
- Length preservation: Decrypted data has same length as original
- Non-identity: Encrypted data differs from plaintext (except empty strings)
