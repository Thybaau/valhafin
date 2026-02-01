package price

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// CurrencyConverter handles currency conversion
type CurrencyConverter struct {
	client *http.Client
	cache  *ExchangeRateCache
}

// ExchangeRateCache caches exchange rates
type ExchangeRateCache struct {
	mu         sync.RWMutex
	rates      map[string]float64 // e.g., "USD_EUR" -> 0.92
	ttl        time.Duration
	lastUpdate time.Time
}

// NewCurrencyConverter creates a new currency converter
func NewCurrencyConverter() *CurrencyConverter {
	return &CurrencyConverter{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: &ExchangeRateCache{
			rates: make(map[string]float64),
			ttl:   1 * time.Hour, // Cache rates for 1 hour
		},
	}
}

// Convert converts an amount from one currency to another
func (c *CurrencyConverter) Convert(amount float64, from, to string) (float64, error) {
	if from == to {
		return amount, nil
	}

	rate, err := c.GetExchangeRate(from, to)
	if err != nil {
		return 0, err
	}

	return amount * rate, nil
}

// GetExchangeRate gets the exchange rate from one currency to another
func (c *CurrencyConverter) GetExchangeRate(from, to string) (float64, error) {
	key := fmt.Sprintf("%s_%s", from, to)

	// Check cache
	if rate := c.cache.Get(key); rate > 0 {
		return rate, nil
	}

	// Fetch from API (using exchangerate-api.com free tier)
	url := fmt.Sprintf("https://api.exchangerate-api.com/v4/latest/%s", from)

	resp, err := c.client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch exchange rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("exchange rate API returned status %d", resp.StatusCode)
	}

	var result struct {
		Rates map[string]float64 `json:"rates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to parse exchange rate response: %w", err)
	}

	rate, ok := result.Rates[to]
	if !ok {
		return 0, fmt.Errorf("exchange rate not found for %s to %s", from, to)
	}

	// Cache the rate
	c.cache.Set(key, rate)

	return rate, nil
}

// Get retrieves a rate from cache if not expired
func (c *ExchangeRateCache) Get(key string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Since(c.lastUpdate) > c.ttl {
		return 0
	}

	return c.rates[key]
}

// Set stores a rate in cache
func (c *ExchangeRateCache) Set(key string, rate float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rates[key] = rate
	c.lastUpdate = time.Now()
}
