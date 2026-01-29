package models

import (
	"errors"
	"time"
)

// Account represents a financial account on a trading platform
type Account struct {
	ID          string     `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Platform    string     `json:"platform" db:"platform"` // "traderepublic", "binance", "boursedirect"
	Credentials string     `json:"-" db:"credentials"`     // Encrypted credentials
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	LastSync    *time.Time `json:"last_sync,omitempty" db:"last_sync"`
}

// Validate validates the Account model
func (a *Account) Validate() error {
	if a.Name == "" {
		return errors.New("account name is required")
	}

	if a.Platform == "" {
		return errors.New("platform is required")
	}

	// Validate platform is one of the supported platforms
	validPlatforms := map[string]bool{
		"traderepublic": true,
		"binance":       true,
		"boursedirect":  true,
	}

	if !validPlatforms[a.Platform] {
		return errors.New("platform must be one of: traderepublic, binance, boursedirect")
	}

	if a.Credentials == "" {
		return errors.New("credentials are required")
	}

	return nil
}
