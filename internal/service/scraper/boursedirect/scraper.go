package boursedirect

import (
	"net/http"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/service/scraper/types"
)

// Scraper implements the scraper.Scraper interface for Bourse Direct
type Scraper struct {
	client *http.Client
}

// NewScraper creates a new Bourse Direct scraper
func NewScraper() *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetPlatformName returns the platform identifier
func (s *Scraper) GetPlatformName() string {
	return "boursedirect"
}

// ValidateCredentials checks if the provided credentials are valid for Bourse Direct
func (s *Scraper) ValidateCredentials(credentials map[string]interface{}) error {
	username, ok := credentials["username"].(string)
	if !ok || username == "" {
		return types.NewValidationError("boursedirect", "username is required", nil)
	}

	password, ok := credentials["password"].(string)
	if !ok || password == "" {
		return types.NewValidationError("boursedirect", "password is required", nil)
	}

	return nil
}

// FetchTransactions retrieves transactions from Bourse Direct
func (s *Scraper) FetchTransactions(credentials map[string]interface{}, lastSync *time.Time) ([]models.Transaction, error) {
	// Validate credentials first
	if err := s.ValidateCredentials(credentials); err != nil {
		return nil, err
	}

	// TODO: Implement Bourse Direct integration
	// Options:
	// 1. CSV import from manual exports
	// 2. Web scraping (reverse engineering)
	// 3. Third-party aggregator API

	// For now, return empty list as stub
	return []models.Transaction{}, types.NewValidationError("boursedirect", "Bourse Direct scraper not yet implemented", nil)
}

// ImportFromCSV imports transactions from exported CSV file
func (s *Scraper) ImportFromCSV(filepath string) ([]models.Transaction, error) {
	// TODO: Implement CSV parser for Bourse Direct format
	return nil, types.NewValidationError("boursedirect", "CSV import not yet implemented", nil)
}
