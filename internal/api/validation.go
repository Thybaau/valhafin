package api

import (
	"fmt"
	"regexp"
)

// CredentialsValidator validates credentials for different platforms
type CredentialsValidator struct{}

// NewCredentialsValidator creates a new credentials validator
func NewCredentialsValidator() *CredentialsValidator {
	return &CredentialsValidator{}
}

// ValidateCredentials validates credentials based on the platform
func (v *CredentialsValidator) ValidateCredentials(platform string, credentials map[string]interface{}) error {
	switch platform {
	case "traderepublic":
		return v.validateTradeRepublicCredentials(credentials)
	case "binance":
		return v.validateBinanceCredentials(credentials)
	case "boursedirect":
		return v.validateBourseDirectCredentials(credentials)
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}

// validateTradeRepublicCredentials validates Trade Republic credentials
func (v *CredentialsValidator) validateTradeRepublicCredentials(credentials map[string]interface{}) error {
	// Trade Republic requires phone_number and pin
	phoneNumber, ok := credentials["phone_number"].(string)
	if !ok || phoneNumber == "" {
		return fmt.Errorf("phone_number is required for Trade Republic")
	}

	// Validate phone number format (international format)
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(phoneNumber) {
		return fmt.Errorf("phone_number must be in international format (e.g., +33612345678)")
	}

	pin, ok := credentials["pin"].(string)
	if !ok || pin == "" {
		return fmt.Errorf("pin is required for Trade Republic")
	}

	// Validate PIN format (4 digits)
	pinRegex := regexp.MustCompile(`^\d{4}$`)
	if !pinRegex.MatchString(pin) {
		return fmt.Errorf("pin must be exactly 4 digits")
	}

	return nil
}

// validateBinanceCredentials validates Binance credentials
func (v *CredentialsValidator) validateBinanceCredentials(credentials map[string]interface{}) error {
	// Binance requires api_key and api_secret
	apiKey, ok := credentials["api_key"].(string)
	if !ok || apiKey == "" {
		return fmt.Errorf("api_key is required for Binance")
	}

	// Validate API key format (64 characters alphanumeric)
	if len(apiKey) != 64 {
		return fmt.Errorf("api_key must be exactly 64 characters")
	}

	apiKeyRegex := regexp.MustCompile(`^[A-Za-z0-9]{64}$`)
	if !apiKeyRegex.MatchString(apiKey) {
		return fmt.Errorf("api_key must contain only alphanumeric characters")
	}

	apiSecret, ok := credentials["api_secret"].(string)
	if !ok || apiSecret == "" {
		return fmt.Errorf("api_secret is required for Binance")
	}

	// Validate API secret format (64 characters alphanumeric)
	if len(apiSecret) != 64 {
		return fmt.Errorf("api_secret must be exactly 64 characters")
	}

	apiSecretRegex := regexp.MustCompile(`^[A-Za-z0-9]{64}$`)
	if !apiSecretRegex.MatchString(apiSecret) {
		return fmt.Errorf("api_secret must contain only alphanumeric characters")
	}

	return nil
}

// validateBourseDirectCredentials validates Bourse Direct credentials
func (v *CredentialsValidator) validateBourseDirectCredentials(credentials map[string]interface{}) error {
	// Bourse Direct requires username and password
	username, ok := credentials["username"].(string)
	if !ok || username == "" {
		return fmt.Errorf("username is required for Bourse Direct")
	}

	// Validate username format (alphanumeric, 3-50 characters)
	if len(username) < 3 || len(username) > 50 {
		return fmt.Errorf("username must be between 3 and 50 characters")
	}

	usernameRegex := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username must contain only alphanumeric characters, underscores, and hyphens")
	}

	password, ok := credentials["password"].(string)
	if !ok || password == "" {
		return fmt.Errorf("password is required for Bourse Direct")
	}

	// Validate password format (minimum 8 characters)
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	return nil
}
