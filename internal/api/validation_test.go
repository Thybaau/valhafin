package api

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestValidateTradeRepublicCredentials tests Trade Republic credential validation
func TestValidateTradeRepublicCredentials(t *testing.T) {
	validator := NewCredentialsValidator()

	tests := []struct {
		name        string
		credentials map[string]interface{}
		wantErr     bool
	}{
		{
			name: "valid credentials",
			credentials: map[string]interface{}{
				"phone_number": "+33612345678",
				"pin":          "1234",
			},
			wantErr: false,
		},
		{
			name: "invalid phone number format",
			credentials: map[string]interface{}{
				"phone_number": "0612345678",
				"pin":          "1234",
			},
			wantErr: true,
		},
		{
			name: "invalid PIN - not 4 digits",
			credentials: map[string]interface{}{
				"phone_number": "+33612345678",
				"pin":          "12",
			},
			wantErr: true,
		},
		{
			name: "missing phone_number",
			credentials: map[string]interface{}{
				"pin": "1234",
			},
			wantErr: true,
		},
		{
			name: "missing pin",
			credentials: map[string]interface{}{
				"phone_number": "+33612345678",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateTradeRepublicCredentials(tt.credentials)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTradeRepublicCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateBinanceCredentials tests Binance credential validation
func TestValidateBinanceCredentials(t *testing.T) {
	validator := NewCredentialsValidator()

	tests := []struct {
		name        string
		credentials map[string]interface{}
		wantErr     bool
	}{
		{
			name: "valid credentials",
			credentials: map[string]interface{}{
				"api_key":    "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01",
				"api_secret": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01",
			},
			wantErr: false,
		},
		{
			name: "invalid api_key length",
			credentials: map[string]interface{}{
				"api_key":    "short",
				"api_secret": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01",
			},
			wantErr: true,
		},
		{
			name: "missing api_secret",
			credentials: map[string]interface{}{
				"api_key": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateBinanceCredentials(tt.credentials)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBinanceCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateBourseDirectCredentials tests Bourse Direct credential validation
func TestValidateBourseDirectCredentials(t *testing.T) {
	validator := NewCredentialsValidator()

	tests := []struct {
		name        string
		credentials map[string]interface{}
		wantErr     bool
	}{
		{
			name: "valid credentials",
			credentials: map[string]interface{}{
				"username": "testuser",
				"password": "testpassword123",
			},
			wantErr: false,
		},
		{
			name: "username too short",
			credentials: map[string]interface{}{
				"username": "ab",
				"password": "testpassword123",
			},
			wantErr: true,
		},
		{
			name: "password too short",
			credentials: map[string]interface{}{
				"username": "testuser",
				"password": "short",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateBourseDirectCredentials(tt.credentials)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBourseDirectCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// **Propriété 2: Rejet des identifiants invalides (Unit Test Version)**
// **Valide: Exigences 1.4**
//
// Property-based test for credential validation without database dependency
func TestProperty_CredentialValidation(t *testing.T) {
	validator := NewCredentialsValidator()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Property: Valid Trade Republic credentials should always pass validation
	properties.Property("valid Trade Republic credentials pass validation", prop.ForAll(
		func(phoneDigits string, pin string) bool {
			// Construct valid phone number and PIN
			phoneNumber := "+33" + phoneDigits
			validPin := pin[:4] // Take first 4 digits

			credentials := map[string]interface{}{
				"phone_number": phoneNumber,
				"pin":          validPin,
			}

			err := validator.validateTradeRepublicCredentials(credentials)
			return err == nil
		},
		gen.RegexMatch(`[0-9]{9}`),  // 9 digits for phone
		gen.RegexMatch(`[0-9]{4,}`), // At least 4 digits for PIN
	))

	// Property: Invalid phone numbers should always fail validation
	properties.Property("invalid phone numbers fail validation", prop.ForAll(
		func(invalidPhone string) bool {
			credentials := map[string]interface{}{
				"phone_number": invalidPhone,
				"pin":          "1234",
			}

			err := validator.validateTradeRepublicCredentials(credentials)
			return err != nil
		},
		gen.OneConstOf("invalid", "0612345678", "12345", ""),
	))

	// Property: Invalid PINs should always fail validation
	properties.Property("invalid PINs fail validation", prop.ForAll(
		func(invalidPin string) bool {
			credentials := map[string]interface{}{
				"phone_number": "+33612345678",
				"pin":          invalidPin,
			}

			err := validator.validateTradeRepublicCredentials(credentials)
			return err != nil
		},
		gen.OneConstOf("12", "123", "12345", "abcd", ""),
	))

	// Property: Valid Binance credentials should always pass validation
	properties.Property("valid Binance credentials pass validation", prop.ForAll(
		func(seed int) bool {
			// Generate valid 64-character alphanumeric strings
			apiKey := generateAlphanumeric(64, seed)
			apiSecret := generateAlphanumeric(64, seed+1)

			credentials := map[string]interface{}{
				"api_key":    apiKey,
				"api_secret": apiSecret,
			}

			err := validator.validateBinanceCredentials(credentials)
			return err == nil
		},
		gen.IntRange(0, 1000000),
	))

	// Property: Valid Bourse Direct credentials should always pass validation
	properties.Property("valid Bourse Direct credentials pass validation", prop.ForAll(
		func(username string, password string) bool {
			// Ensure valid format
			if len(username) < 3 || len(username) > 50 {
				return true // Skip invalid inputs
			}
			if len(password) < 8 {
				return true // Skip invalid inputs
			}

			credentials := map[string]interface{}{
				"username": username,
				"password": password,
			}

			err := validator.validateBourseDirectCredentials(credentials)
			return err == nil
		},
		gen.RegexMatch(`[A-Za-z0-9_-]{3,50}`),
		gen.RegexMatch(`[A-Za-z0-9!@#$%^&*]{8,}`),
	))

	properties.TestingRun(t)
}

// Helper function to generate alphanumeric strings
func generateAlphanumeric(length int, seed int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[(seed+i)%len(charset)]
	}
	return string(result)
}
