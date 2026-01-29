package types

import (
	"time"
	"valhafin/internal/domain/models"
)

// Scraper defines the common interface for all platform scrapers
type Scraper interface {
	// FetchTransactions retrieves transactions from the platform
	// If lastSync is nil, it performs a full sync (all historical transactions)
	// If lastSync is provided, it performs an incremental sync (only new transactions)
	FetchTransactions(credentials map[string]interface{}, lastSync *time.Time) ([]models.Transaction, error)

	// ValidateCredentials checks if the provided credentials are valid for the platform
	ValidateCredentials(credentials map[string]interface{}) error

	// GetPlatformName returns the platform identifier
	GetPlatformName() string
}

// SyncResult contains the result of a synchronization operation
type SyncResult struct {
	AccountID           string    `json:"account_id"`
	Platform            string    `json:"platform"`
	TransactionsFetched int       `json:"transactions_fetched"`
	TransactionsStored  int       `json:"transactions_stored"`
	SyncType            string    `json:"sync_type"` // "full" or "incremental"
	StartTime           time.Time `json:"start_time"`
	EndTime             time.Time `json:"end_time"`
	Duration            string    `json:"duration"`
	Error               string    `json:"error,omitempty"`
}

// ScraperError represents an error that occurred during scraping
type ScraperError struct {
	Platform string
	Type     string // "auth", "network", "parsing", "validation"
	Message  string
	Retry    bool
	Err      error
}

func (e *ScraperError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *ScraperError) Unwrap() error {
	return e.Err
}

// NewAuthError creates a new authentication error
func NewAuthError(platform, message string, err error) *ScraperError {
	return &ScraperError{
		Platform: platform,
		Type:     "auth",
		Message:  message,
		Retry:    false,
		Err:      err,
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(platform, message string, err error) *ScraperError {
	return &ScraperError{
		Platform: platform,
		Type:     "network",
		Message:  message,
		Retry:    true,
		Err:      err,
	}
}

// NewParsingError creates a new parsing error
func NewParsingError(platform, message string, err error) *ScraperError {
	return &ScraperError{
		Platform: platform,
		Type:     "parsing",
		Message:  message,
		Retry:    false,
		Err:      err,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(platform, message string, err error) *ScraperError {
	return &ScraperError{
		Platform: platform,
		Type:     "validation",
		Message:  message,
		Retry:    false,
		Err:      err,
	}
}
