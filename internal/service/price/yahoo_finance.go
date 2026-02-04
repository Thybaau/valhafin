package price

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"
)

// YahooFinanceService implements the Service interface using Yahoo Finance API
type YahooFinanceService struct {
	db                *database.DB
	httpClient        *http.Client
	cache             *PriceCache
	currencyConverter *CurrencyConverter
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
			ttl:    1 * time.Hour,
		},
		currencyConverter: NewCurrencyConverter(),
	}
}

// GetCurrentPrice retrieves the current price for an asset by ISIN
func (s *YahooFinanceService) GetCurrentPrice(isin string) (*models.AssetPrice, error) {
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

	// Fetch price from Yahoo Finance
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

	symbol := ""
	if asset.Symbol != nil {
		symbol = *asset.Symbol
	}

	if symbol == "" {
		return nil, fmt.Errorf("no symbol found for asset %s", isin)
	}

	// Determine range based on date difference
	daysDiff := endDate.Sub(startDate).Hours() / 24
	var rangeStr string
	var interval string

	if daysDiff <= 7 {
		rangeStr = "5d"
		interval = "1d"
	} else if daysDiff <= 30 {
		rangeStr = "1mo"
		interval = "1d"
	} else if daysDiff <= 90 {
		rangeStr = "3mo"
		interval = "1d"
	} else if daysDiff <= 180 {
		rangeStr = "6mo"
		interval = "1d"
	} else if daysDiff <= 365 {
		rangeStr = "1y"
		interval = "1d"
	} else if daysDiff <= 730 {
		rangeStr = "2y"
		interval = "1wk"
	} else if daysDiff <= 1825 {
		rangeStr = "5y"
		interval = "1wk"
	} else {
		rangeStr = "max"
		interval = "1mo"
	}

	historicalPrices, err := s.fetchHistoricalPrices(symbol, isin, asset.Currency, rangeStr, interval)
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
		// Small delay to be respectful to Yahoo Finance
		time.Sleep(100 * time.Millisecond)
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
func (s *YahooFinanceService) fetchAndStorePrice(isin, symbol, expectedCurrency string) (*models.AssetPrice, error) {
	// Fetch from Yahoo Finance
	price, currency, err := s.fetchPriceFromYahoo(symbol)
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

// fetchPriceFromYahoo fetches the current price from Yahoo Finance API
func (s *YahooFinanceService) fetchPriceFromYahoo(symbol string) (float64, string, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?range=1d&interval=1m", symbol)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add User-Agent to avoid rate limiting
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

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
	var result YahooChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if result.Chart.Error != nil {
		return 0, "", fmt.Errorf("Yahoo Finance error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return 0, "", fmt.Errorf("no data available for symbol %s", symbol)
	}

	chartResult := result.Chart.Result[0]

	// Get current price from meta
	price := chartResult.Meta.RegularMarketPrice
	if price == 0 {
		return 0, "", fmt.Errorf("no price data available")
	}

	currency := chartResult.Meta.Currency

	return price, currency, nil
}

// fetchHistoricalPrices fetches historical prices from Yahoo Finance
func (s *YahooFinanceService) fetchHistoricalPrices(symbol, isin, expectedCurrency, rangeStr, interval string) ([]models.AssetPrice, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?range=%s&interval=%s", symbol, rangeStr, interval)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add User-Agent to avoid rate limiting
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Yahoo Finance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Yahoo Finance returned status %d", resp.StatusCode)
	}

	var result YahooChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Chart.Error != nil {
		return nil, fmt.Errorf("Yahoo Finance error: %s", result.Chart.Error.Description)
	}

	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data available")
	}

	return s.parseChartData(result.Chart.Result[0], isin, expectedCurrency)
}

// parseChartData parses Yahoo Finance chart data and converts currency
func (s *YahooFinanceService) parseChartData(chartResult YahooChartResult, isin, expectedCurrency string) ([]models.AssetPrice, error) {
	var prices []models.AssetPrice

	sourceCurrency := chartResult.Meta.Currency

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

	// Extract timestamps and close prices
	timestamps := chartResult.Timestamp
	if len(chartResult.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no quote data available")
	}

	closePrices := chartResult.Indicators.Quote[0].Close

	for i, timestamp := range timestamps {
		if i >= len(closePrices) {
			break
		}

		closePrice := closePrices[i]
		if closePrice == nil {
			continue
		}

		// Convert currency
		finalPrice := *closePrice * exchangeRate
		finalCurrency := expectedCurrency

		prices = append(prices, models.AssetPrice{
			ISIN:      isin,
			Price:     finalPrice,
			Currency:  finalCurrency,
			Timestamp: time.Unix(int64(timestamp), 0),
		})
	}

	return prices, nil
}

// Yahoo Finance API response structures

type YahooChartResponse struct {
	Chart struct {
		Result []YahooChartResult `json:"result"`
		Error  *YahooError        `json:"error"`
	} `json:"chart"`
}

type YahooError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type YahooChartResult struct {
	Meta       YahooMeta       `json:"meta"`
	Timestamp  []int           `json:"timestamp"`
	Indicators YahooIndicators `json:"indicators"`
}

type YahooMeta struct {
	Currency           string  `json:"currency"`
	Symbol             string  `json:"symbol"`
	RegularMarketPrice float64 `json:"regularMarketPrice"`
	ChartPreviousClose float64 `json:"chartPreviousClose"`
	PreviousClose      float64 `json:"previousClose"`
}

type YahooIndicators struct {
	Quote []YahooQuote `json:"quote"`
}

type YahooQuote struct {
	Open   []*float64 `json:"open"`
	High   []*float64 `json:"high"`
	Low    []*float64 `json:"low"`
	Close  []*float64 `json:"close"`
	Volume []*int64   `json:"volume"`
}

// YahooSearchResult represents a search result from Yahoo Finance
type YahooSearchResult struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"longname"`
	ShortName string  `json:"shortname"`
	Type      string  `json:"quoteType"`
	TypeDisp  string  `json:"typeDisp"`
	Exchange  string  `json:"exchange"`
	ExchDisp  string  `json:"exchDisp"`
	Sector    string  `json:"sector"`
	Industry  string  `json:"industry"`
	Score     float64 `json:"score"`
}

// SearchSymbol searches for symbols on Yahoo Finance
func (s *YahooFinanceService) SearchSymbol(query string) ([]YahooSearchResult, error) {
	// URL encode the query
	encodedQuery := url.QueryEscape(query)
	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v1/finance/search?q=%s&quotesCount=15&newsCount=0", encodedQuery)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add User-Agent to avoid rate limiting
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := s.httpClient.Do(req)
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
		return nil, fmt.Errorf("Yahoo Finance API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var response struct {
		Quotes []YahooSearchResult `json:"quotes"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Quotes, nil
}

// ResolveSymbolWithExchange resolves a symbol to its full Yahoo Finance symbol with exchange suffix
// Uses Trade Republic exchange information to select the best match
func (s *YahooFinanceService) ResolveSymbolWithExchange(symbol string, trExchanges []string, assetName string) (string, bool, error) {
	// Search for the symbol on Yahoo Finance
	results, err := s.SearchSymbol(symbol)
	if err != nil {
		return "", false, fmt.Errorf("failed to search symbol: %w", err)
	}

	if len(results) == 0 {
		// Try searching by asset name as fallback
		if assetName != "" {
			results, err = s.SearchSymbol(assetName)
			if err != nil || len(results) == 0 {
				return "", false, fmt.Errorf("no results found for symbol %s or name %s", symbol, assetName)
			}
		} else {
			return "", false, fmt.Errorf("no results found for symbol %s", symbol)
		}
	}

	// Mapping Trade Republic exchanges to Yahoo Finance exchanges
	exchangeMapping := map[string]string{
		"LSX":   "LSE", // London Stock Exchange
		"GER":   "GER", // Xetra
		"XETR":  "GER", // Xetra
		"XETRA": "GER", // Xetra
		"XFRA":  "GER", // Frankfurt (use Xetra)
		"PAR":   "PAR", // Euronext Paris
		"XPAR":  "PAR", // Euronext Paris
		"MIL":   "MIL", // Borsa Italiana
		"XMIL":  "MIL", // Borsa Italiana
		"AMS":   "AMS", // Euronext Amsterdam
		"SWX":   "SWX", // Swiss Exchange
		"XSWX":  "SWX", // Swiss Exchange
		"TDG":   "GER", // Tradegate (use Xetra as fallback)
		"TIB":   "GER", // Tradegate (use Xetra as fallback)
	}

	// Exchange priority (lower = better)
	// Prioritize EUR exchanges for European ETFs
	exchangePriority := map[string]int{
		"GER":   1, // Xetra (EUR, high liquidity)
		"XETRA": 1, // Xetra
		"PAR":   2, // Paris (EUR)
		"AMS":   3, // Amsterdam (EUR)
		"MIL":   4, // Milan (EUR)
		"LSE":   5, // London (GBP/USD, less priority for EUR assets)
		"SWX":   6, // Swiss (CHF)
	}

	// Method 1: Try to match with Trade Republic exchange, prioritizing EUR exchanges
	if len(trExchanges) > 0 {
		// First pass: look for EUR exchanges (GER, PAR, etc.)
		for _, trExch := range trExchanges {
			yahooExch := exchangeMapping[trExch]
			if yahooExch != "" && (yahooExch == "GER" || yahooExch == "PAR" || yahooExch == "AMS" || yahooExch == "MIL") {
				for _, result := range results {
					if result.Exchange == yahooExch {
						// Validate that the symbol works
						if s.validateSymbol(result.Symbol) {
							log.Printf("INFO: Resolved %s to %s (matched EUR exchange %s)", symbol, result.Symbol, yahooExch)
							return result.Symbol, true, nil
						}
					}
				}
			}
		}

		// Second pass: try any matching exchange
		for _, trExch := range trExchanges {
			yahooExch := exchangeMapping[trExch]
			if yahooExch != "" {
				for _, result := range results {
					if result.Exchange == yahooExch {
						// Validate that the symbol works
						if s.validateSymbol(result.Symbol) {
							log.Printf("INFO: Resolved %s to %s (matched exchange %s)", symbol, result.Symbol, yahooExch)
							return result.Symbol, true, nil
						}
					}
				}
			}
		}
	}

	// Method 2: Use exchange priority
	var bestResult *YahooSearchResult
	bestPriority := 999

	for i := range results {
		priority, exists := exchangePriority[results[i].Exchange]
		if exists && priority < bestPriority {
			bestPriority = priority
			bestResult = &results[i]
		}
	}

	if bestResult != nil {
		// Validate that the symbol works
		if s.validateSymbol(bestResult.Symbol) {
			log.Printf("INFO: Resolved %s to %s (priority-based)", symbol, bestResult.Symbol)
			return bestResult.Symbol, true, nil
		}
	}

	// Method 3: Use highest score from Yahoo
	if len(results) > 0 {
		// Sort by score descending
		bestScore := results[0]
		for i := range results {
			if results[i].Score > bestScore.Score {
				bestScore = results[i]
			}
		}

		// Validate that the symbol works
		if s.validateSymbol(bestScore.Symbol) {
			log.Printf("INFO: Resolved %s to %s (score-based)", symbol, bestScore.Symbol)
			return bestScore.Symbol, true, nil
		}
	}

	// If all methods fail, return the first result without validation
	if len(results) > 0 {
		log.Printf("WARNING: Could not validate symbol for %s, using first result %s", symbol, results[0].Symbol)
		return results[0].Symbol, false, nil
	}

	return "", false, fmt.Errorf("could not resolve symbol %s", symbol)
}

// validateSymbol checks if a symbol exists and has price data on Yahoo Finance
func (s *YahooFinanceService) validateSymbol(symbol string) bool {
	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?range=1d&interval=1d", symbol)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	var result YahooChartResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}

	// Check if we got valid data
	return result.Chart.Error == nil && len(result.Chart.Result) > 0
}
