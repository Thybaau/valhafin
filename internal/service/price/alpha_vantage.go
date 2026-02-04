package price

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"
)

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

// AlphaVantageService implements the Service interface using Alpha Vantage API
type AlphaVantageService struct {
	db                *database.DB
	httpClient        *http.Client
	apiKey            string
	cache             *PriceCache
	currencyConverter *CurrencyConverter
}

// NewAlphaVantageService creates a new Alpha Vantage price service
func NewAlphaVantageService(db *database.DB, apiKey string) *AlphaVantageService {
	return &AlphaVantageService{
		db:     db,
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: &PriceCache{
			prices: make(map[string]*CachedPrice),
			ttl:    1 * time.Hour,
		},
		currencyConverter: NewCurrencyConverter(),
	}
}

// GetCurrentPrice retrieves the current price for an asset by ISIN
func (s *AlphaVantageService) GetCurrentPrice(isin string) (*models.AssetPrice, error) {
	log.Printf("DEBUG: GetCurrentPrice for ISIN %s", isin)

	// Check cache first
	if cachedPrice := s.cache.Get(isin); cachedPrice != nil {
		log.Printf("DEBUG: Returning cached price for %s", isin)
		return cachedPrice, nil
	}

	// Get asset from database to retrieve symbol
	asset, err := s.db.GetAssetByISIN(isin)
	if err != nil {
		log.Printf("DEBUG: Asset not found in DB for %s", isin)
		// Fallback: try to get last known price from database
		lastPrice, dbErr := s.db.GetLatestAssetPrice(isin)
		if dbErr == nil {
			s.cache.Set(isin, lastPrice)
			return lastPrice, nil
		}
		return nil, fmt.Errorf("asset not found and no fallback available: %w", err)
	}

	// Get symbol from asset
	symbol := ""
	if asset.Symbol != nil {
		symbol = *asset.Symbol
	}

	if symbol == "" {
		return nil, fmt.Errorf("no symbol found for asset %s", isin)
	}

	log.Printf("DEBUG: Asset found for %s, symbol: %s, currency: %s", isin, symbol, asset.Currency)

	// Fetch price from Alpha Vantage
	price, err := s.fetchAndStorePrice(isin, symbol, asset.Currency)
	if err != nil {
		log.Printf("DEBUG: Failed to fetch price for %s: %v", isin, err)
		// Fallback: try to get last known price from database
		lastPrice, dbErr := s.db.GetLatestAssetPrice(isin)
		if dbErr == nil {
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
func (s *AlphaVantageService) GetPriceHistory(isin string, startDate, endDate time.Time) ([]models.AssetPrice, error) {
	// First, try to get from database
	prices, err := s.db.GetAssetPriceHistory(isin, startDate, endDate)
	if err == nil && len(prices) > 0 {
		return prices, nil
	}

	// If not in database or empty, fetch from Alpha Vantage
	asset, err := s.db.GetAssetByISIN(isin)
	if err != nil {
		return nil, fmt.Errorf("asset not found: %w", err)
	}

	symbol := ""
	if asset.Symbol != nil {
		symbol = *asset.Symbol
	}

	if symbol == "" {
		return nil, fmt.Errorf("no symbol found for asset %s", isin)
	}

	// Determine which function to use based on date range
	daysDiff := endDate.Sub(startDate).Hours() / 24

	var historicalPrices []models.AssetPrice

	if daysDiff <= 100 {
		// Use daily data for short periods
		historicalPrices, err = s.fetchDailyPrices(symbol, isin, asset.Currency)
	} else if daysDiff <= 1825 { // ~5 years
		// Use weekly data for medium periods
		historicalPrices, err = s.fetchWeeklyPrices(symbol, isin, asset.Currency)
	} else {
		// Use monthly data for long periods
		historicalPrices, err = s.fetchMonthlyPrices(symbol, isin, asset.Currency)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical prices: %w", err)
	}

	// Filter by date range
	var filteredPrices []models.AssetPrice
	for _, price := range historicalPrices {
		if (price.Timestamp.Equal(startDate) || price.Timestamp.After(startDate)) &&
			(price.Timestamp.Equal(endDate) || price.Timestamp.Before(endDate)) {
			filteredPrices = append(filteredPrices, price)
		}
	}

	// Store in database
	if len(filteredPrices) > 0 {
		if err := s.db.CreateAssetPricesBatch(filteredPrices); err != nil {
			log.Printf("Warning: failed to store historical prices: %v", err)
		}
	}

	return filteredPrices, nil
}

// UpdateAllPrices updates prices for all assets in the database
func (s *AlphaVantageService) UpdateAllPrices() error {
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
		// Rate limiting: Alpha Vantage allows 5 requests per minute
		time.Sleep(12 * time.Second)
	}

	if len(errors) > 0 && successCount == 0 {
		return fmt.Errorf("failed to update all prices: %d errors", len(errors))
	}

	return nil
}

// UpdateAssetPrice updates the price for a specific asset
func (s *AlphaVantageService) UpdateAssetPrice(isin string) error {
	_, err := s.GetCurrentPrice(isin)
	return err
}

// fetchAndStorePrice fetches the current price from Alpha Vantage and stores it
func (s *AlphaVantageService) fetchAndStorePrice(isin, symbol, expectedCurrency string) (*models.AssetPrice, error) {
	// Fetch from Alpha Vantage
	price, currency, err := s.fetchPriceFromAlphaVantage(symbol)
	if err != nil {
		return nil, err
	}

	// Convert currency if needed
	if currency != expectedCurrency {
		convertedPrice, err := s.currencyConverter.Convert(price, currency, expectedCurrency)
		if err != nil {
			log.Printf("Warning: failed to convert %s to %s for ISIN %s: %v", currency, expectedCurrency, isin, err)
		} else {
			log.Printf("Converted price for %s: %.2f %s -> %.2f %s", isin, price, currency, convertedPrice, expectedCurrency)
			price = convertedPrice
			currency = expectedCurrency
		}
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

// fetchPriceFromAlphaVantage fetches the current price from Alpha Vantage API
func (s *AlphaVantageService) fetchPriceFromAlphaVantage(symbol string) (float64, string, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", symbol, s.apiKey)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return 0, "", fmt.Errorf("failed to fetch from Alpha Vantage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", fmt.Errorf("Alpha Vantage returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result AlphaVantageQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if result.Note != "" {
		return 0, "", fmt.Errorf("Alpha Vantage API limit: %s", result.Note)
	}

	if result.ErrorMessage != "" {
		return 0, "", fmt.Errorf("Alpha Vantage error: %s", result.ErrorMessage)
	}

	if result.GlobalQuote.Price == "" {
		return 0, "", fmt.Errorf("no price data available")
	}

	// Parse price
	var price float64
	if _, err := fmt.Sscanf(result.GlobalQuote.Price, "%f", &price); err != nil {
		return 0, "", fmt.Errorf("failed to parse price: %w", err)
	}

	// Determine currency from symbol (approximation)
	currency := "USD" // Default
	if len(symbol) > 2 && symbol[len(symbol)-2:] == ".L" {
		currency = "GBP" // London Stock Exchange
	}

	return price, currency, nil
}

// fetchDailyPrices fetches daily historical prices
func (s *AlphaVantageService) fetchDailyPrices(symbol, isin, expectedCurrency string) ([]models.AssetPrice, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&apikey=%s", symbol, s.apiKey)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Alpha Vantage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Alpha Vantage returned status %d", resp.StatusCode)
	}

	var result AlphaVantageDailyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Note != "" {
		return nil, fmt.Errorf("Alpha Vantage API limit: %s", result.Note)
	}

	return s.parseTimeSeriesData(result.TimeSeries, isin, expectedCurrency, symbol)
}

// fetchWeeklyPrices fetches weekly historical prices
func (s *AlphaVantageService) fetchWeeklyPrices(symbol, isin, expectedCurrency string) ([]models.AssetPrice, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_WEEKLY&symbol=%s&apikey=%s", symbol, s.apiKey)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Alpha Vantage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Alpha Vantage returned status %d", resp.StatusCode)
	}

	var result AlphaVantageWeeklyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Note != "" {
		return nil, fmt.Errorf("Alpha Vantage API limit: %s", result.Note)
	}

	return s.parseTimeSeriesData(result.WeeklyTimeSeries, isin, expectedCurrency, symbol)
}

// fetchMonthlyPrices fetches monthly historical prices
func (s *AlphaVantageService) fetchMonthlyPrices(symbol, isin, expectedCurrency string) ([]models.AssetPrice, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_MONTHLY&symbol=%s&apikey=%s", symbol, s.apiKey)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Alpha Vantage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Alpha Vantage returned status %d", resp.StatusCode)
	}

	var result AlphaVantageMonthlyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Note != "" {
		return nil, fmt.Errorf("Alpha Vantage API limit: %s", result.Note)
	}

	return s.parseTimeSeriesData(result.MonthlyTimeSeries, isin, expectedCurrency, symbol)
}

// parseTimeSeriesData parses time series data and converts currency
func (s *AlphaVantageService) parseTimeSeriesData(timeSeries map[string]TimeSeriesData, isin, expectedCurrency, symbol string) ([]models.AssetPrice, error) {
	var prices []models.AssetPrice

	// Determine source currency
	sourceCurrency := "USD"
	if len(symbol) > 2 && symbol[len(symbol)-2:] == ".L" {
		sourceCurrency = "GBP"
	}

	// Get exchange rate once for all prices
	exchangeRate := 1.0
	var err error
	if sourceCurrency != expectedCurrency {
		exchangeRate, err = s.currencyConverter.GetExchangeRate(sourceCurrency, expectedCurrency)
		if err != nil {
			log.Printf("Warning: failed to get exchange rate %s to %s: %v", sourceCurrency, expectedCurrency, err)
			exchangeRate = 1.0
		}
	}

	for dateStr, data := range timeSeries {
		// Parse date
		timestamp, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("Warning: failed to parse date %s: %v", dateStr, err)
			continue
		}

		// Parse close price
		var closePrice float64
		if _, err := fmt.Sscanf(data.Close, "%f", &closePrice); err != nil {
			log.Printf("Warning: failed to parse price %s: %v", data.Close, err)
			continue
		}

		// Convert currency
		finalPrice := closePrice * exchangeRate
		finalCurrency := expectedCurrency

		prices = append(prices, models.AssetPrice{
			ISIN:      isin,
			Price:     finalPrice,
			Currency:  finalCurrency,
			Timestamp: timestamp,
		})
	}

	return prices, nil
}

// Alpha Vantage API response structures

type AlphaVantageQuoteResponse struct {
	GlobalQuote struct {
		Symbol           string `json:"01. symbol"`
		Open             string `json:"02. open"`
		High             string `json:"03. high"`
		Low              string `json:"04. low"`
		Price            string `json:"05. price"`
		Volume           string `json:"06. volume"`
		LatestTradingDay string `json:"07. latest trading day"`
		PreviousClose    string `json:"08. previous close"`
		Change           string `json:"09. change"`
		ChangePercent    string `json:"10. change percent"`
	} `json:"Global Quote"`
	Note         string `json:"Note"`
	ErrorMessage string `json:"Error Message"`
}

type TimeSeriesData struct {
	Open   string `json:"1. open"`
	High   string `json:"2. high"`
	Low    string `json:"3. low"`
	Close  string `json:"4. close"`
	Volume string `json:"5. volume"`
}

type AlphaVantageDailyResponse struct {
	MetaData   map[string]string         `json:"Meta Data"`
	TimeSeries map[string]TimeSeriesData `json:"Time Series (Daily)"`
	Note       string                    `json:"Note"`
}

type AlphaVantageWeeklyResponse struct {
	MetaData         map[string]string         `json:"Meta Data"`
	WeeklyTimeSeries map[string]TimeSeriesData `json:"Weekly Time Series"`
	Note             string                    `json:"Note"`
}

type AlphaVantageMonthlyResponse struct {
	MetaData          map[string]string         `json:"Meta Data"`
	MonthlyTimeSeries map[string]TimeSeriesData `json:"Monthly Time Series"`
	Note              string                    `json:"Note"`
}

// AlphaVantageSearchResult represents a search result from Alpha Vantage
type AlphaVantageSearchResult struct {
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Region      string `json:"region"`
	Currency    string `json:"currency"`
	MatchScore  string `json:"match_score"`
	MarketOpen  string `json:"market_open"`
	MarketClose string `json:"market_close"`
	Timezone    string `json:"timezone"`
}

// SearchSymbol searches for symbols on Alpha Vantage
func (s *AlphaVantageService) SearchSymbol(query string) ([]AlphaVantageSearchResult, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=SYMBOL_SEARCH&keywords=%s&apikey=%s",
		query, s.apiKey)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbol: %w", err)
	}
	defer resp.Body.Close()

	// Read the body first
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		// Try to extract error message from body
		var errorResponse struct {
			Information  string `json:"Information"`
			Note         string `json:"Note"`
			ErrorMessage string `json:"Error Message"`
		}
		if err := json.Unmarshal(body, &errorResponse); err == nil {
			if errorResponse.Information != "" {
				return nil, fmt.Errorf("%s", errorResponse.Information)
			}
			if errorResponse.Note != "" {
				return nil, fmt.Errorf("%s", errorResponse.Note)
			}
			if errorResponse.ErrorMessage != "" {
				return nil, fmt.Errorf("%s", errorResponse.ErrorMessage)
			}
		}
		return nil, fmt.Errorf("Alpha Vantage API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Check for Alpha Vantage error messages (they sometimes return 200 with error in body)
	var errorResponse struct {
		Information  string `json:"Information"`
		Note         string `json:"Note"`
		ErrorMessage string `json:"Error Message"`
	}

	if err := json.Unmarshal(body, &errorResponse); err == nil {
		if errorResponse.Information != "" {
			return nil, fmt.Errorf("Alpha Vantage: %s", errorResponse.Information)
		}
		if errorResponse.Note != "" {
			return nil, fmt.Errorf("Alpha Vantage: %s", errorResponse.Note)
		}
		if errorResponse.ErrorMessage != "" {
			return nil, fmt.Errorf("Alpha Vantage: %s", errorResponse.ErrorMessage)
		}
	}

	// Parse the actual response
	var response struct {
		BestMatches []map[string]interface{} `json:"bestMatches"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]AlphaVantageSearchResult, 0, len(response.BestMatches))
	for _, match := range response.BestMatches {
		result := AlphaVantageSearchResult{
			Symbol:      getStringValue(match, "1. symbol"),
			Name:        getStringValue(match, "2. name"),
			Type:        getStringValue(match, "3. type"),
			Region:      getStringValue(match, "4. region"),
			Currency:    getStringValue(match, "8. currency"),
			MatchScore:  getStringValue(match, "9. matchScore"),
			MarketOpen:  getStringValue(match, "5. marketOpen"),
			MarketClose: getStringValue(match, "6. marketClose"),
			Timezone:    getStringValue(match, "7. timezone"),
		}
		results = append(results, result)
	}

	return results, nil
}

// getStringValue safely extracts a string value from a map
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
