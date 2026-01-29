package binance

import (
	"net/http"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/service/scraper/types"
)

// Scraper implements the scraper.Scraper interface for Binance
type Scraper struct {
	client *http.Client
}

// NewScraper creates a new Binance scraper
func NewScraper() *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetPlatformName returns the platform identifier
func (s *Scraper) GetPlatformName() string {
	return "binance"
}

// ValidateCredentials checks if the provided credentials are valid for Binance
func (s *Scraper) ValidateCredentials(credentials map[string]interface{}) error {
	apiKey, ok := credentials["api_key"].(string)
	if !ok || apiKey == "" {
		return types.NewValidationError("binance", "api_key is required", nil)
	}

	apiSecret, ok := credentials["api_secret"].(string)
	if !ok || apiSecret == "" {
		return types.NewValidationError("binance", "api_secret is required", nil)
	}

	// Validate API key format (basic check)
	if len(apiKey) < 32 {
		return types.NewValidationError("binance", "Invalid API key format", nil)
	}

	if len(apiSecret) < 32 {
		return types.NewValidationError("binance", "Invalid API secret format", nil)
	}

	return nil
}

// FetchTransactions retrieves transactions from Binance
func (s *Scraper) FetchTransactions(credentials map[string]interface{}, lastSync *time.Time) ([]models.Transaction, error) {
	// Validate credentials first
	if err := s.ValidateCredentials(credentials); err != nil {
		return nil, err
	}

	// TODO: Implement Binance API integration
	// This would involve:
	// 1. Using the Binance REST API with API key/secret
	// 2. Fetching account trades, deposits, withdrawals
	// 3. Converting Binance data format to our Transaction model
	// 4. Filtering by lastSync if provided (incremental sync)

	// For now, return empty list as stub
	return []models.Transaction{}, types.NewValidationError("binance", "Binance scraper not yet implemented", nil)
}
