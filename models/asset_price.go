package models

import (
	"errors"
	"time"
)

// AssetPrice represents the price of an asset at a specific time
type AssetPrice struct {
	ID        int64     `json:"id" db:"id"`
	ISIN      string    `json:"isin" db:"isin"`
	Price     float64   `json:"price" db:"price"`
	Currency  string    `json:"currency" db:"currency"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

// Validate validates the AssetPrice model
func (ap *AssetPrice) Validate() error {
	if ap.ISIN == "" {
		return errors.New("ISIN is required")
	}

	if ap.Price <= 0 {
		return errors.New("price must be greater than 0")
	}

	if ap.Currency == "" {
		return errors.New("currency is required")
	}

	if ap.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}

	return nil
}
