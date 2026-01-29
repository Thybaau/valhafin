package price

import (
	"time"
	"valhafin/internal/domain/models"
)

// Service defines the interface for price retrieval operations
type Service interface {
	// GetCurrentPrice retrieves the current price for an asset by ISIN
	GetCurrentPrice(isin string) (*models.AssetPrice, error)

	// GetPriceHistory retrieves historical prices for an asset within a date range
	GetPriceHistory(isin string, startDate, endDate time.Time) ([]models.AssetPrice, error)

	// UpdateAllPrices updates prices for all assets in the database
	UpdateAllPrices() error

	// UpdateAssetPrice updates the price for a specific asset
	UpdateAssetPrice(isin string) error
}
