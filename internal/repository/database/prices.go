package database

import (
	"fmt"
	"time"
	"valhafin/internal/domain/models"
)

// CreateAsset creates a new asset in the database
func (db *DB) CreateAsset(asset *models.Asset) error {
	// Validate asset
	if err := asset.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Set last updated timestamp
	asset.LastUpdated = time.Now()

	query := `
		INSERT INTO assets (isin, name, symbol, type, currency, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (isin) DO UPDATE
		SET name = EXCLUDED.name,
		    symbol = EXCLUDED.symbol,
		    type = EXCLUDED.type,
		    currency = EXCLUDED.currency,
		    last_updated = EXCLUDED.last_updated
	`

	_, err := db.Exec(
		query,
		asset.ISIN,
		asset.Name,
		asset.Symbol,
		asset.Type,
		asset.Currency,
		asset.LastUpdated,
	)

	if err != nil {
		return fmt.Errorf("failed to create asset: %w", err)
	}

	return nil
}

// GetAssetByISIN retrieves an asset by its ISIN
func (db *DB) GetAssetByISIN(isin string) (*models.Asset, error) {
	var asset models.Asset

	query := `
		SELECT isin, name, symbol, type, currency, last_updated
		FROM assets
		WHERE isin = $1
	`

	err := db.Get(&asset, query, isin)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	return &asset, nil
}

// GetAllAssets retrieves all assets
func (db *DB) GetAllAssets() ([]models.Asset, error) {
	var assets []models.Asset

	query := `
		SELECT isin, name, symbol, type, currency, last_updated
		FROM assets
		ORDER BY name
	`

	err := db.Select(&assets, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get assets: %w", err)
	}

	return assets, nil
}

// GetAssetsByType retrieves all assets of a specific type
func (db *DB) GetAssetsByType(assetType string) ([]models.Asset, error) {
	var assets []models.Asset

	query := `
		SELECT isin, name, symbol, type, currency, last_updated
		FROM assets
		WHERE type = $1
		ORDER BY name
	`

	err := db.Select(&assets, query, assetType)
	if err != nil {
		return nil, fmt.Errorf("failed to get assets by type: %w", err)
	}

	return assets, nil
}

// UpdateAsset updates an existing asset
func (db *DB) UpdateAsset(asset *models.Asset) error {
	// Validate asset
	if err := asset.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Update last updated timestamp
	asset.LastUpdated = time.Now()

	query := `
		UPDATE assets
		SET name = $1, symbol = $2, type = $3, currency = $4, last_updated = $5
		WHERE isin = $6
	`

	result, err := db.Exec(
		query,
		asset.Name,
		asset.Symbol,
		asset.Type,
		asset.Currency,
		asset.LastUpdated,
		asset.ISIN,
	)

	if err != nil {
		return fmt.Errorf("failed to update asset: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("asset not found")
	}

	return nil
}

// DeleteAsset deletes an asset
func (db *DB) DeleteAsset(isin string) error {
	query := `DELETE FROM assets WHERE isin = $1`

	result, err := db.Exec(query, isin)
	if err != nil {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("asset not found")
	}

	return nil
}

// CreateAssetPrice creates a new asset price record
func (db *DB) CreateAssetPrice(price *models.AssetPrice) error {
	// Validate price
	if err := price.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO asset_prices (isin, price, currency, timestamp)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (isin, timestamp) DO UPDATE
		SET price = EXCLUDED.price,
		    currency = EXCLUDED.currency
		RETURNING id
	`

	err := db.Get(&price.ID, query, price.ISIN, price.Price, price.Currency, price.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create asset price: %w", err)
	}

	return nil
}

// CreateAssetPricesBatch creates multiple asset prices in a single transaction
func (db *DB) CreateAssetPricesBatch(prices []models.AssetPrice) error {
	if len(prices) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO asset_prices (isin, price, currency, timestamp)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (isin, timestamp) DO UPDATE
		SET price = EXCLUDED.price,
		    currency = EXCLUDED.currency
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, price := range prices {
		if err := price.Validate(); err != nil {
			return fmt.Errorf("validation failed for price: %w", err)
		}

		_, err := stmt.Exec(price.ISIN, price.Price, price.Currency, price.Timestamp)
		if err != nil {
			return fmt.Errorf("failed to insert price: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetLatestAssetPrice retrieves the most recent price for an asset
func (db *DB) GetLatestAssetPrice(isin string) (*models.AssetPrice, error) {
	var price models.AssetPrice

	query := `
		SELECT id, isin, price, currency, timestamp
		FROM asset_prices
		WHERE isin = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`

	err := db.Get(&price, query, isin)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest price: %w", err)
	}

	return &price, nil
}

// GetAssetPriceHistory retrieves price history for an asset within a date range
func (db *DB) GetAssetPriceHistory(isin string, startDate, endDate time.Time) ([]models.AssetPrice, error) {
	var prices []models.AssetPrice

	query := `
		SELECT id, isin, price, currency, timestamp
		FROM asset_prices
		WHERE isin = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp ASC
	`

	err := db.Select(&prices, query, isin, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get price history: %w", err)
	}

	return prices, nil
}

// GetAssetPriceAt retrieves the price of an asset at or before a specific time
func (db *DB) GetAssetPriceAt(isin string, timestamp time.Time) (*models.AssetPrice, error) {
	var price models.AssetPrice

	query := `
		SELECT id, isin, price, currency, timestamp
		FROM asset_prices
		WHERE isin = $1 AND timestamp <= $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	err := db.Get(&price, query, isin, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get price at timestamp: %w", err)
	}

	return &price, nil
}

// GetAllLatestPrices retrieves the latest price for all assets
func (db *DB) GetAllLatestPrices() ([]models.AssetPrice, error) {
	var prices []models.AssetPrice

	query := `
		SELECT DISTINCT ON (isin) id, isin, price, currency, timestamp
		FROM asset_prices
		ORDER BY isin, timestamp DESC
	`

	err := db.Select(&prices, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all latest prices: %w", err)
	}

	return prices, nil
}

// DeleteOldPrices deletes price records older than a specified date
func (db *DB) DeleteOldPrices(beforeDate time.Time) (int64, error) {
	query := `DELETE FROM asset_prices WHERE timestamp < $1`

	result, err := db.Exec(query, beforeDate)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old prices: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
