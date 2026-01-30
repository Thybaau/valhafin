package models

import (
	"errors"
	"regexp"
	"time"
)

// Asset represents a financial asset (stock, ETF, crypto)
type Asset struct {
	ISIN        string    `json:"isin" db:"isin"`
	Name        string    `json:"name" db:"name"`
	Symbol      *string   `json:"symbol,omitempty" db:"symbol"`
	Type        string    `json:"type" db:"type"` // "stock", "etf", "crypto"
	Currency    string    `json:"currency" db:"currency"`
	LastUpdated time.Time `json:"last_updated" db:"last_updated"`
}

// Validate validates the Asset model
func (a *Asset) Validate() error {
	if a.ISIN == "" {
		return errors.New("ISIN is required")
	}

	// Validate ISIN format (12 characters: 2 letters + 10 alphanumeric)
	isinRegex := regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{10}$`)
	if !isinRegex.MatchString(a.ISIN) {
		return errors.New("ISIN must be 12 characters: 2 letters followed by 10 alphanumeric characters")
	}

	if a.Name == "" {
		return errors.New("asset name is required")
	}

	if a.Type == "" {
		return errors.New("asset type is required")
	}

	// Validate asset type
	validTypes := map[string]bool{
		"stock":  true,
		"etf":    true,
		"crypto": true,
	}

	if !validTypes[a.Type] {
		return errors.New("asset type must be one of: stock, etf, crypto")
	}

	if a.Currency == "" {
		return errors.New("currency is required")
	}

	// Validate currency format (3 uppercase letters)
	currencyRegex := regexp.MustCompile(`^[A-Z]{3}$`)
	if !currencyRegex.MatchString(a.Currency) {
		return errors.New("currency must be 3 uppercase letters (e.g., USD, EUR)")
	}

	return nil
}
