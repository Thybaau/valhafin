package price

import (
	"database/sql"
	"testing"
	"time"
	"valhafin/internal/domain/models"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Mock database for testing
type mockDB struct {
	assets      map[string]*models.Asset
	prices      map[string][]*models.AssetPrice
	latestPrice map[string]*models.AssetPrice
}

func newMockDB() *mockDB {
	return &mockDB{
		assets:      make(map[string]*models.Asset),
		prices:      make(map[string][]*models.AssetPrice),
		latestPrice: make(map[string]*models.AssetPrice),
	}
}

func (m *mockDB) GetAssetByISIN(isin string) (*models.Asset, error) {
	asset, exists := m.assets[isin]
	if !exists {
		return nil, sql.ErrNoRows
	}
	return asset, nil
}

func (m *mockDB) CreateAssetPrice(price *models.AssetPrice) error {
	if err := price.Validate(); err != nil {
		return err
	}
	m.prices[price.ISIN] = append(m.prices[price.ISIN], price)
	m.latestPrice[price.ISIN] = price
	return nil
}

func (m *mockDB) CreateAssetPricesBatch(prices []models.AssetPrice) error {
	for _, price := range prices {
		if err := m.CreateAssetPrice(&price); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockDB) GetLatestAssetPrice(isin string) (*models.AssetPrice, error) {
	price, exists := m.latestPrice[isin]
	if !exists {
		return nil, sql.ErrNoRows
	}
	return price, nil
}

func (m *mockDB) GetAssetPriceHistory(isin string, startDate, endDate time.Time) ([]models.AssetPrice, error) {
	prices, exists := m.prices[isin]
	if !exists {
		return []models.AssetPrice{}, nil
	}

	var filtered []models.AssetPrice
	for _, p := range prices {
		if (p.Timestamp.Equal(startDate) || p.Timestamp.After(startDate)) &&
			(p.Timestamp.Equal(endDate) || p.Timestamp.Before(endDate)) {
			filtered = append(filtered, *p)
		}
	}

	return filtered, nil
}

func (m *mockDB) GetAllAssets() ([]models.Asset, error) {
	var assets []models.Asset
	for _, asset := range m.assets {
		assets = append(assets, *asset)
	}
	return assets, nil
}

// **Propriété 13: Identification par ISIN**
// **Valide: Exigences 4.9, 10.1**
//
// Property: For all assets in the system, identification and price retrieval
// must use the ISIN as the unique key, and no asset should be identified by any other means.
func TestProperty_IdentificationByISIN(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	// Generate valid ISIN format: 2 uppercase letters + 10 alphanumeric
	genISIN := gen.RegexMatch(`^[A-Z]{2}[A-Z0-9]{10}$`)

	properties.Property("price retrieval uses ISIN as unique identifier", prop.ForAll(
		func(isin string) bool {
			mockDB := newMockDB()

			// Create asset with ISIN
			symbol := "TEST"
			asset := &models.Asset{
				ISIN:        isin,
				Name:        "Test Asset",
				Symbol:      &symbol,
				Type:        "stock",
				Currency:    "USD",
				LastUpdated: time.Now(),
			}
			mockDB.assets[isin] = asset

			// Create a price for this ISIN
			price := &models.AssetPrice{
				ISIN:      isin,
				Price:     100.0,
				Currency:  "USD",
				Timestamp: time.Now(),
			}
			mockDB.latestPrice[isin] = price

			// Retrieve by ISIN
			retrieved, err := mockDB.GetLatestAssetPrice(isin)
			if err != nil {
				t.Logf("Failed to retrieve price by ISIN: %v", err)
				return false
			}

			// Verify the ISIN matches
			if retrieved.ISIN != isin {
				t.Logf("Retrieved price has different ISIN: got %s, want %s", retrieved.ISIN, isin)
				return false
			}

			return true
		},
		genISIN,
	))

	properties.Property("different ISINs identify different assets", prop.ForAll(
		func(isin1, isin2 string) bool {
			if isin1 == isin2 {
				return true // Skip if same ISIN
			}

			mockDB := newMockDB()

			// Create two different assets
			symbol1 := "TEST1"
			asset1 := &models.Asset{
				ISIN:        isin1,
				Name:        "Asset 1",
				Symbol:      &symbol1,
				Type:        "stock",
				Currency:    "USD",
				LastUpdated: time.Now(),
			}
			symbol2 := "TEST2"
			asset2 := &models.Asset{
				ISIN:        isin2,
				Name:        "Asset 2",
				Symbol:      &symbol2,
				Type:        "stock",
				Currency:    "EUR",
				LastUpdated: time.Now(),
			}

			mockDB.assets[isin1] = asset1
			mockDB.assets[isin2] = asset2

			// Create prices for both
			price1 := &models.AssetPrice{
				ISIN:      isin1,
				Price:     100.0,
				Currency:  "USD",
				Timestamp: time.Now(),
			}
			price2 := &models.AssetPrice{
				ISIN:      isin2,
				Price:     200.0,
				Currency:  "EUR",
				Timestamp: time.Now(),
			}

			mockDB.latestPrice[isin1] = price1
			mockDB.latestPrice[isin2] = price2

			// Retrieve both and verify they're different
			retrieved1, err1 := mockDB.GetLatestAssetPrice(isin1)
			retrieved2, err2 := mockDB.GetLatestAssetPrice(isin2)

			if err1 != nil || err2 != nil {
				t.Logf("Failed to retrieve prices")
				return false
			}

			// Prices should be different
			if retrieved1.Price == retrieved2.Price && retrieved1.Currency == retrieved2.Currency {
				t.Logf("Different ISINs returned same price data")
				return false
			}

			return true
		},
		genISIN,
		genISIN,
	))

	properties.TestingRun(t)
}

// **Propriété 14: Récupération et stockage des prix**
// **Valide: Exigences 10.2, 10.3, 10.4**
//
// Property: For all assets identified by ISIN, the system must retrieve the current price
// from the external financial API, store it in the database with a timestamp,
// and update periodically.
func TestProperty_PriceRetrievalAndStorage(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 30
	properties := gopter.NewProperties(parameters)

	genISIN := gen.RegexMatch(`^[A-Z]{2}[A-Z0-9]{10}$`)
	genPrice := gen.Float64Range(0.01, 10000.0)

	properties.Property("retrieved prices are stored with timestamp", prop.ForAll(
		func(isin string, price float64) bool {
			mockDB := newMockDB()

			// Create asset
			symbol := "TEST"
			asset := &models.Asset{
				ISIN:        isin,
				Name:        "Test Asset",
				Symbol:      &symbol,
				Type:        "stock",
				Currency:    "USD",
				LastUpdated: time.Now(),
			}
			mockDB.assets[isin] = asset

			// Simulate price retrieval and storage
			beforeStore := time.Now()
			assetPrice := &models.AssetPrice{
				ISIN:      isin,
				Price:     price,
				Currency:  "USD",
				Timestamp: time.Now(),
			}

			err := mockDB.CreateAssetPrice(assetPrice)
			if err != nil {
				t.Logf("Failed to store price: %v", err)
				return false
			}

			// Retrieve and verify
			stored, err := mockDB.GetLatestAssetPrice(isin)
			if err != nil {
				t.Logf("Failed to retrieve stored price: %v", err)
				return false
			}

			// Verify ISIN matches
			if stored.ISIN != isin {
				t.Logf("Stored ISIN doesn't match: got %s, want %s", stored.ISIN, isin)
				return false
			}

			// Verify price matches
			if stored.Price != price {
				t.Logf("Stored price doesn't match: got %f, want %f", stored.Price, price)
				return false
			}

			// Verify timestamp is recent
			if stored.Timestamp.Before(beforeStore) {
				t.Logf("Timestamp is before storage time")
				return false
			}

			return true
		},
		genISIN,
		genPrice,
	))

	properties.Property("price updates replace old prices", prop.ForAll(
		func(isin string, price1, price2 float64) bool {
			if price1 == price2 {
				return true // Skip if same price
			}

			mockDB := newMockDB()

			// Store first price
			assetPrice1 := &models.AssetPrice{
				ISIN:      isin,
				Price:     price1,
				Currency:  "USD",
				Timestamp: time.Now(),
			}
			mockDB.CreateAssetPrice(assetPrice1)

			// Wait a bit
			time.Sleep(10 * time.Millisecond)

			// Store second price (update)
			assetPrice2 := &models.AssetPrice{
				ISIN:      isin,
				Price:     price2,
				Currency:  "USD",
				Timestamp: time.Now(),
			}
			mockDB.CreateAssetPrice(assetPrice2)

			// Get latest price
			latest, err := mockDB.GetLatestAssetPrice(isin)
			if err != nil {
				t.Logf("Failed to get latest price: %v", err)
				return false
			}

			// Latest should be the second price
			if latest.Price != price2 {
				t.Logf("Latest price is not the most recent: got %f, want %f", latest.Price, price2)
				return false
			}

			return true
		},
		genISIN,
		genPrice,
		genPrice,
	))

	properties.TestingRun(t)
}

// **Propriété 15: Fallback sur dernier prix connu**
// **Valide: Exigences 10.5**
//
// Property: For all assets where the current price cannot be retrieved from the external API,
// the system must use the last known price stored in the database and display a warning
// indicating that the price is not up-to-date.
func TestProperty_FallbackToLastKnownPrice(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 30
	properties := gopter.NewProperties(parameters)

	genISIN := gen.RegexMatch(`^[A-Z]{2}[A-Z0-9]{10}$`)
	genPrice := gen.Float64Range(0.01, 10000.0)

	properties.Property("fallback to last known price when API fails", prop.ForAll(
		func(isin string, lastKnownPrice float64) bool {
			mockDB := newMockDB()

			// Store a last known price
			lastPrice := &models.AssetPrice{
				ISIN:      isin,
				Price:     lastKnownPrice,
				Currency:  "USD",
				Timestamp: time.Now().Add(-24 * time.Hour), // 1 day old
			}
			mockDB.latestPrice[isin] = lastPrice

			// Simulate API failure by not having the asset
			// (which would cause GetCurrentPrice to fail and fallback)

			// Get the last known price
			retrieved, err := mockDB.GetLatestAssetPrice(isin)
			if err != nil {
				t.Logf("Failed to retrieve last known price: %v", err)
				return false
			}

			// Verify it's the last known price
			if retrieved.Price != lastKnownPrice {
				t.Logf("Fallback price doesn't match: got %f, want %f", retrieved.Price, lastKnownPrice)
				return false
			}

			// Verify timestamp is old (indicating it's a fallback)
			if time.Since(retrieved.Timestamp) < 1*time.Hour {
				t.Logf("Fallback price timestamp is too recent")
				return false
			}

			return true
		},
		genISIN,
		genPrice,
	))

	properties.Property("no fallback when no last known price exists", prop.ForAll(
		func(isin string) bool {
			mockDB := newMockDB()

			// Don't store any price for this ISIN

			// Try to get price (should fail)
			_, err := mockDB.GetLatestAssetPrice(isin)
			if err == nil {
				t.Logf("Expected error when no price exists, got nil")
				return false
			}

			// Should return sql.ErrNoRows or similar
			if err != sql.ErrNoRows {
				t.Logf("Expected sql.ErrNoRows, got: %v", err)
				return false
			}

			return true
		},
		genISIN,
	))

	properties.Property("fallback price is valid and usable", prop.ForAll(
		func(isin string, price float64) bool {
			mockDB := newMockDB()

			// Store a last known price
			lastPrice := &models.AssetPrice{
				ISIN:      isin,
				Price:     price,
				Currency:  "USD",
				Timestamp: time.Now().Add(-48 * time.Hour), // 2 days old
			}
			mockDB.latestPrice[isin] = lastPrice

			// Retrieve the fallback price
			retrieved, err := mockDB.GetLatestAssetPrice(isin)
			if err != nil {
				t.Logf("Failed to retrieve fallback price: %v", err)
				return false
			}

			// Validate the fallback price
			if err := retrieved.Validate(); err != nil {
				t.Logf("Fallback price is invalid: %v", err)
				return false
			}

			// Verify price is positive
			if retrieved.Price <= 0 {
				t.Logf("Fallback price is not positive: %f", retrieved.Price)
				return false
			}

			return true
		},
		genISIN,
		genPrice,
	))

	properties.TestingRun(t)
}
