package traderepublic

import (
	"net/http"
	"valhafin/config"
)

const (
	baseURL   = "https://api.traderepublic.com"
	wsURL     = "wss://api.traderepublic.com"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
)

type Scraper struct {
	config *config.Config
	client *http.Client
}

func NewScraper(cfg *config.Config) *Scraper {
	return &Scraper{
		config: cfg,
		client: &http.Client{},
	}
}
