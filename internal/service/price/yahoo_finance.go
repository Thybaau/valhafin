package price

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"
)

// YahooFinanceService implements the Service interface using Yahoo Finance API
type YahooFinanceService struct {
	db         *database.DB
	httpClient *http.Client
	cache      *PriceCache
	isinMapper *ISINMapper
}

// PriceCache provides in-memory caching for prices
type PriceCache struct {
	mu     sync.RWMutex
	prices map[string]*CachedPrice
	ttl    time.Duration
}

// CachedPrice represents a cached price with expiration
type CachedPrice struct {
	Price     *models.AssetPrice
	ExpiresAt time.Time
}

// ISINMapper handles ISIN to Yahoo Finance symbol conversion
type ISINMapper struct {
	mu      sync.RWMutex
	mapping map[string]string // ISIN -> Yahoo Symbol
}

// NewYahooFinanceService creates a new Yahoo Finance price service
func NewYahooFinanceService(db *database.DB) *YahooFinanceService {
	return &YahooFinanceService{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: &PriceCache{
			prices: make(map[string]*CachedPrice),
			ttl:    1 * time.Hour, // Cache for 1 hour
		},
		isinMapper: &ISINMapper{
			mapping: make(map[string]string),
		},
	}
}

// GetCurrentPrice retrieves the current price for an asset by ISIN
func (s *YahooFinanceService) GetCurrentPrice(isin string) (*models.AssetPrice, error) {
	// Check cache first
	if cachedPrice := s.cache.Get(isin); cachedPrice != nil {
		return cachedPrice, nil
	}

	// Get asset from database to retrieve symbol
	asset, err := s.db.GetAssetByISIN(isin)
	if err != nil {
		// If asset not found, try to fetch price anyway using ISIN conversion
		return s.fetchAndStorePrice(isin, "")
	}

	// Fetch price from Yahoo Finance
	price, err := s.fetchAndStorePrice(isin, asset.Symbol)
	if err != nil {
		// Fallback: try to get last known price from database
		lastPrice, dbErr := s.db.GetLatestAssetPrice(isin)
		if dbErr == nil {
			// Cache the fallback price
			s.cache.Set(isin, lastPrice)
			return lastPrice, nil
		}
		return nil, fmt.Errorf("failed to fetch price and no fallback available: %w", err)
	}

	// Cache the new price
	s.cache.Set(isin, price)

	return price, nil
}

// GetPriceHistory retrieves historical prices for an asset within a date range
func (s *YahooFinanceService) GetPriceHistory(isin string, startDate, endDate time.Time) ([]models.AssetPrice, error) {
	// First, try to get from database
	prices, err := s.db.GetAssetPriceHistory(isin, startDate, endDate)
	if err == nil && len(prices) > 0 {
		return prices, nil
	}

	// If not in database or empty, fetch from Yahoo Finance
	asset, err := s.db.GetAssetByISIN(isin)
	if err != nil {
		return nil, fmt.Errorf("asset not found: %w", err)
	}

	// Convert ISIN to Yahoo symbol
	symbol := s.isinMapper.GetSymbol(isin, asset.Symbol)
	if symbol == "" {
		symbol = s.convertISINToSymbol(isin)
	}

	// Fetch historical data from Yahoo Finance
	historicalPrices, err := s.fetchHistoricalPrices(symbol, isin, asset.Currency, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical prices: %w", err)
	}

	// Store in database
	if len(historicalPrices) > 0 {
		if err := s.db.CreateAssetPricesBatch(historicalPrices); err != nil {
			// Log error but don't fail - we still have the data
			fmt.Printf("Warning: failed to store historical prices: %v\n", err)
		}
	}

	return historicalPrices, nil
}

// UpdateAllPrices updates prices for all assets in the database
func (s *YahooFinanceService) UpdateAllPrices() error {
	assets, err := s.db.GetAllAssets()
	if err != nil {
		return fmt.Errorf("failed to get assets: %w", err)
	}

	var errors []error
	successCount := 0

	for _, asset := range assets {
		if err := s.UpdateAssetPrice(asset.ISIN); err != nil {
			errors = append(errors, fmt.Errorf("failed to update %s: %w", asset.ISIN, err))
		} else {
			successCount++
		}
	}

	if len(errors) > 0 && successCount == 0 {
		return fmt.Errorf("failed to update all prices: %d errors", len(errors))
	}

	return nil
}

// UpdateAssetPrice updates the price for a specific asset
func (s *YahooFinanceService) UpdateAssetPrice(isin string) error {
	_, err := s.GetCurrentPrice(isin)
	return err
}

// fetchAndStorePrice fetches the current price from Yahoo Finance and stores it
func (s *YahooFinanceService) fetchAndStorePrice(isin, symbol string) (*models.AssetPrice, error) {
	// Convert ISIN to Yahoo symbol if not provided
	if symbol == "" {
		symbol = s.isinMapper.GetSymbol(isin, "")
		if symbol == "" {
			symbol = s.convertISINToSymbol(isin)
		}
	}

	// Fetch from Yahoo Finance
	price, currency, err := s.fetchPriceFromYahoo(symbol)
	if err != nil {
		return nil, err
	}

	// Create asset price model
	assetPrice := &models.AssetPrice{
		ISIN:      isin,
		Price:     price,
		Currency:  currency,
		Timestamp: time.Now(),
	}

	// Store in database
	if err := s.db.CreateAssetPrice(assetPrice); err != nil {
		return nil, fmt.Errorf("failed to store price: %w", err)
	}

	return assetPrice, nil
}

// fetchPriceFromYahoo fetches the current price from Yahoo Finance API
func (s *YahooFinanceService) fetchPriceFromYahoo(symbol string) (float64, string, error) {
	// Yahoo Finance v8 API endpoint
	baseURL := "https://query1.finance.yahoo.com/v8/finance/chart/" + url.PathEscape(symbol)

	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Valhafin/1.0)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to fetch from Yahoo Finance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", fmt.Errorf("Yahoo Finance returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result YahooFinanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract price and currency
	if result.Chart.Error != nil {
		return 0, "", fmt.Errorf("Yahoo Finance error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return 0, "", fmt.Errorf("no data returned from Yahoo Finance")
	}

	chartResult := result.Chart.Result[0]
	if chartResult.Meta.RegularMarketPrice == 0 {
		return 0, "", fmt.Errorf("no price data available")
	}

	return chartResult.Meta.RegularMarketPrice, chartResult.Meta.Currency, nil
}

// fetchHistoricalPrices fetches historical prices from Yahoo Finance
func (s *YahooFinanceService) fetchHistoricalPrices(symbol, isin, currency string, startDate, endDate time.Time) ([]models.AssetPrice, error) {
	// Convert dates to Unix timestamps
	period1 := startDate.Unix()
	period2 := endDate.Unix()

	// Yahoo Finance v8 API endpoint for historical data
	baseURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d",
		url.PathEscape(symbol), period1, period2)

	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Valhafin/1.0)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Yahoo Finance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Yahoo Finance returned status %d", resp.StatusCode)
	}

	// Parse response
	var result YahooFinanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Chart.Error != nil {
		return nil, fmt.Errorf("Yahoo Finance error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data returned from Yahoo Finance")
	}

	chartResult := result.Chart.Result[0]

	// Extract timestamps and closing prices
	timestamps := chartResult.Timestamp
	closes := chartResult.Indicators.Quote[0].Close

	if len(timestamps) != len(closes) {
		return nil, fmt.Errorf("mismatched data lengths")
	}

	// Build price history
	var prices []models.AssetPrice
	for i := 0; i < len(timestamps); i++ {
		if closes[i] == nil {
			continue // Skip nil prices
		}

		prices = append(prices, models.AssetPrice{
			ISIN:      isin,
			Price:     *closes[i],
			Currency:  currency,
			Timestamp: time.Unix(timestamps[i], 0),
		})
	}

	return prices, nil
}

// convertISINToSymbol converts an ISIN to a Yahoo Finance symbol
// This is a simplified conversion - in production, you'd want a more comprehensive mapping
func (s *YahooFinanceService) convertISINToSymbol(isin string) string {
	if len(isin) < 2 {
		return isin
	}

	// Extract country code
	countryCode := isin[:2]

	// Basic conversion rules
	switch countryCode {
	case "US":
		// US stocks: remove country code and checksum
		if len(isin) == 12 {
			return isin[2:11]
		}
	case "DE":
		// German stocks: add .DE suffix
		if len(isin) == 12 {
			return isin[2:11] + ".DE"
		}
	case "FR":
		// French stocks: add .PA suffix
		if len(isin) == 12 {
			return isin[2:11] + ".PA"
		}
	case "GB":
		// UK stocks: add .L suffix
		if len(isin) == 12 {
			return isin[2:11] + ".L"
		}
	}

	// Default: return ISIN as-is
	return isin
}

// Cache methods

// Get retrieves a price from cache if not expired
func (c *PriceCache) Get(isin string) *models.AssetPrice {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.prices[isin]
	if !exists {
		return nil
	}

	if time.Now().After(cached.ExpiresAt) {
		return nil
	}

	return cached.Price
}

// Set stores a price in cache
func (c *PriceCache) Set(isin string, price *models.AssetPrice) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.prices[isin] = &CachedPrice{
		Price:     price,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Clear removes all cached prices
func (c *PriceCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.prices = make(map[string]*CachedPrice)
}

// ISINMapper methods

// GetSymbol retrieves the Yahoo symbol for an ISIN
func (m *ISINMapper) GetSymbol(isin, fallback string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if symbol, exists := m.mapping[isin]; exists {
		return symbol
	}

	return fallback
}

// SetSymbol stores a mapping from ISIN to Yahoo symbol
func (m *ISINMapper) SetSymbol(isin, symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.mapping[isin] = symbol
}

// Yahoo Finance API response structures

type YahooFinanceResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency           string  `json:"currency"`
				Symbol             string  `json:"symbol"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []*float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}
