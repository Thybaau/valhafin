package sync

import (
	"fmt"
	"valhafin/internal/service/scraper/binance"
	"valhafin/internal/service/scraper/boursedirect"
	"valhafin/internal/service/scraper/traderepublic"
	"valhafin/internal/service/scraper/types"
)

// ScraperFactory creates scrapers for different platforms
type ScraperFactory struct {
	scrapers map[string]types.Scraper
}

// NewScraperFactory creates a new scraper factory
func NewScraperFactory() *ScraperFactory {
	return &ScraperFactory{
		scrapers: map[string]types.Scraper{
			"traderepublic": traderepublic.NewScraper(),
			"binance":       binance.NewScraper(),
			"boursedirect":  boursedirect.NewScraper(),
		},
	}
}

// GetScraper returns the appropriate scraper for the given platform
func (f *ScraperFactory) GetScraper(platform string) (types.Scraper, error) {
	scraper, ok := f.scrapers[platform]
	if !ok {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
	return scraper, nil
}

// GetSupportedPlatforms returns a list of supported platforms
func (f *ScraperFactory) GetSupportedPlatforms() []string {
	platforms := make([]string, 0, len(f.scrapers))
	for platform := range f.scrapers {
		platforms = append(platforms, platform)
	}
	return platforms
}
