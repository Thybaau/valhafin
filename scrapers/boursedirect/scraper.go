package boursedirect

import (
	"valhafin/config"
)

// TODO: Implement Bourse Direct scraper
// Options:
// 1. CSV import from manual exports
// 2. Web scraping (reverse engineering)
// 3. Third-party aggregator API

type Scraper struct {
	config *config.Config
}

func NewScraper(cfg *config.Config) *Scraper {
	return &Scraper{
		config: cfg,
	}
}

// ImportFromCSV imports transactions from exported CSV file
func (s *Scraper) ImportFromCSV(filepath string) ([]interface{}, error) {
	// TODO: Implement CSV parser for Bourse Direct format
	return nil, nil
}

// FetchTransactions scrapes transactions from Bourse Direct website
func (s *Scraper) FetchTransactions() ([]interface{}, error) {
	// TODO: Implement web scraping or API reverse engineering
	return nil, nil
}
