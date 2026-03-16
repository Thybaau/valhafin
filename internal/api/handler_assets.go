package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"
	"valhafin/internal/service/price"

	"github.com/gorilla/mux"
)

// AssetPosition represents a user's position in an asset
type AssetPosition struct {
	ISIN              string     `json:"isin"`
	Name              string     `json:"name"`
	Symbol            string     `json:"symbol,omitempty"`
	SymbolVerified    bool       `json:"symbol_verified"`
	Quantity          float64    `json:"quantity"`
	AverageBuyPrice   float64    `json:"average_buy_price"`
	CurrentPrice      float64    `json:"current_price"`
	CurrentValue      float64    `json:"current_value"`
	TotalInvested     float64    `json:"total_invested"`
	UnrealizedGain    float64    `json:"unrealized_gain"`
	UnrealizedGainPct float64    `json:"unrealized_gain_pct"`
	Currency          string     `json:"currency"`
	Purchases         []Purchase `json:"purchases"`
}

// Purchase represents a buy transaction
type Purchase struct {
	Date     string  `json:"date"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

// GetAssetPriceHandler retrieves the current price for an asset by ISIN
// @Summary Prix actuel d'un actif
// @Description Récupère le prix actuel d'un actif par son code ISIN
// @Tags assets
// @Produce json
// @Param isin path string true "Code ISIN de l'actif"
// @Success 200 {object} models.AssetPrice
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/assets/{isin}/price [get]
func (h *Handler) GetAssetPriceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isin := vars["isin"]

	if isin == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "ISIN is required", nil)
		return
	}

	// Get current price from price service
	price, err := h.PriceService.GetCurrentPrice(isin)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Asset not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "PRICE_ERROR", "Failed to retrieve price", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, price)
}

// GetAssetPriceHistoryHandler retrieves historical prices for an asset
// @Summary Historique des prix d'un actif
// @Description Récupère l'historique des prix pour un actif sur une période donnée
// @Tags assets
// @Produce json
// @Param isin path string true "Code ISIN de l'actif"
// @Param start_date query string false "Date de début (YYYY-MM-DD)"
// @Param end_date query string false "Date de fin (YYYY-MM-DD)"
// @Success 200 {array} models.AssetPrice
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/assets/{isin}/history [get]
func (h *Handler) GetAssetPriceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isin := vars["isin"]

	if isin == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "ISIN is required", nil)
		return
	}

	// Parse query parameters
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	// Default to last 30 days if not specified
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if startDateStr != "" {
		parsed, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid start_date format (use YYYY-MM-DD)", nil)
			return
		}
		startDate = parsed
	}

	if endDateStr != "" {
		parsed, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid end_date format (use YYYY-MM-DD)", nil)
			return
		}
		endDate = parsed
	}

	// Validate date range
	if startDate.After(endDate) {
		respondError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", "start_date must be before end_date", nil)
		return
	}

	// Get price history from price service
	prices, err := h.PriceService.GetPriceHistory(isin, startDate, endDate)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Asset not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "PRICE_ERROR", "Failed to retrieve price history", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, prices)
}

// RefreshAssetPricesHandler forces a refresh of all historical prices for an asset
// @Summary Rafraîchir les prix d'un actif
// @Description Supprime le cache et récupère l'historique complet des prix
// @Tags assets
// @Produce json
// @Param isin path string true "Code ISIN de l'actif"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/assets/{isin}/price/refresh [post]
func (h *Handler) RefreshAssetPricesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isin := vars["isin"]

	if isin == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "ISIN is required", nil)
		return
	}

	log.Printf("INFO: Starting price refresh for %s", isin)

	// Delete all existing prices for this asset to force fresh fetch
	result, err := h.DB.Exec("DELETE FROM asset_prices WHERE isin = $1", isin)
	if err != nil {
		log.Printf("ERROR: Failed to delete prices for %s: %v", isin, err)
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to clear price cache", map[string]string{
			"error": err.Error(),
		})
		return
	}

	rowsDeleted, _ := result.RowsAffected()
	log.Printf("INFO: Cleared %d cached prices for %s", rowsDeleted, isin)

	// Fetch complete price history
	if err := h.fetchCompleteAssetPriceHistory(isin); err != nil {
		log.Printf("ERROR: Failed to fetch price history for %s: %v", isin, err)
		respondError(w, http.StatusInternalServerError, "PRICE_ERROR", "Failed to fetch prices", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Get count of stored prices
	var priceCount int
	if err := h.DB.Get(&priceCount, "SELECT COUNT(*) FROM asset_prices WHERE isin = $1", isin); err != nil {
		priceCount = 0
	}

	// Get date range
	var dateRange struct {
		MinDate time.Time `db:"min_date"`
		MaxDate time.Time `db:"max_date"`
	}
	err = h.DB.Get(&dateRange, "SELECT MIN(timestamp) as min_date, MAX(timestamp) as max_date FROM asset_prices WHERE isin = $1", isin)
	if err != nil {
		log.Printf("WARNING: Failed to get date range for %s: %v", isin, err)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "Prices refreshed successfully",
		"isin":          isin,
		"data_points":   priceCount,
		"deleted_cache": rowsDeleted,
		"date_range": map[string]string{
			"from": dateRange.MinDate.Format("2006-01-02"),
			"to":   dateRange.MaxDate.Format("2006-01-02"),
		},
	})
}

// UpdateSingleAssetPrice forces an update of a single asset price
// @Summary Mettre à jour le prix d'un actif
// @Description Force la mise à jour du prix actuel d'un actif
// @Tags assets
// @Produce json
// @Param isin path string true "Code ISIN de l'actif"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/assets/{isin}/price/update [post]
func (h *Handler) UpdateSingleAssetPrice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isin := vars["isin"]

	if isin == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ISIN", "ISIN is required", nil)
		return
	}

	log.Printf("Updating price for asset %s...", isin)

	// Update price for this asset
	if err := h.PriceService.UpdateAssetPrice(isin); err != nil {
		respondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update price", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Get the updated price
	price, err := h.DB.GetLatestAssetPrice(isin)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve updated price", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Price updated successfully",
		"price":   price,
	})
}

// GetAssetsHandler returns all assets with user positions
// @Summary Lister les actifs avec positions
// @Description Retourne tous les actifs avec les positions de l'utilisateur
// @Tags assets
// @Produce json
// @Success 200 {array} AssetPosition
// @Failure 500 {object} ErrorResponse
// @Router /api/assets [get]
func (h *Handler) GetAssetsHandler(w http.ResponseWriter, r *http.Request) {
	// Get all accounts
	accounts, err := h.DB.GetAllAccounts()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get accounts", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Map to store positions by ISIN
	positionsByISIN := make(map[string]*AssetPosition)

	// Collect all transactions from all accounts
	for _, account := range accounts {
		filter := database.TransactionFilter{}
		transactions, err := h.DB.GetTransactionsByAccount(account.ID, account.Platform, filter)
		if err != nil {
			log.Printf("Warning: failed to get transactions for account %s: %v", account.ID, err)
			continue
		}

		// Process transactions
		for _, tx := range transactions {
			if tx.ISIN == nil || *tx.ISIN == "" {
				continue
			}

			isin := *tx.ISIN

			// Initialize position if not exists
			if _, exists := positionsByISIN[isin]; !exists {
				// Get asset info
				asset, err := h.DB.GetAssetByISIN(isin)
				assetName := "Unknown"
				currency := "EUR"
				symbol := ""
				symbolVerified := false
				if err == nil {
					assetName = asset.Name
					currency = asset.Currency
					if asset.Symbol != nil {
						symbol = *asset.Symbol
					}
					symbolVerified = asset.SymbolVerified
				}

				positionsByISIN[isin] = &AssetPosition{
					ISIN:           isin,
					Name:           assetName,
					Symbol:         symbol,
					SymbolVerified: symbolVerified,
					Currency:       currency,
					Purchases:      []Purchase{},
				}
			}

			position := positionsByISIN[isin]

			// Process based on transaction type
			switch tx.TransactionType {
			case "buy":
				position.Quantity += tx.Quantity
				investedAmount := -tx.AmountValue // AmountValue is negative for buys
				position.TotalInvested += investedAmount

				// Add to purchases list
				position.Purchases = append(position.Purchases, Purchase{
					Date:     tx.Timestamp[:10], // Extract date part
					Quantity: tx.Quantity,
					Price:    investedAmount / tx.Quantity,
				})

			case "sell":
				position.Quantity -= tx.Quantity
				// Reduce invested amount proportionally
				if position.Quantity > 0 {
					avgCost := position.TotalInvested / (position.Quantity + tx.Quantity)
					position.TotalInvested -= avgCost * tx.Quantity
				} else {
					position.TotalInvested = 0
				}
			}
		}
	}

	// Calculate current values and get current prices
	var assets []AssetPosition
	for _, position := range positionsByISIN {
		if position.Quantity <= 0 {
			continue // Skip sold positions
		}

		// Calculate average buy price
		if position.Quantity > 0 {
			position.AverageBuyPrice = position.TotalInvested / position.Quantity
		}

		// Get current price
		currentPrice, err := h.PriceService.GetCurrentPrice(position.ISIN)
		if err != nil {
			log.Printf("Warning: failed to get current price for %s: %v", position.ISIN, err)
			// Use average buy price as fallback
			position.CurrentPrice = position.AverageBuyPrice
		} else {
			position.CurrentPrice = currentPrice.Price
		}

		// Calculate current value and gains
		position.CurrentValue = position.Quantity * position.CurrentPrice
		position.UnrealizedGain = position.CurrentValue - position.TotalInvested
		if position.TotalInvested > 0 {
			position.UnrealizedGainPct = (position.UnrealizedGain / position.TotalInvested) * 100
		}

		assets = append(assets, *position)
	}

	// Sort by current value (descending)
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].CurrentValue > assets[j].CurrentValue
	})

	respondJSON(w, http.StatusOK, assets)
}

// SymbolSearchHandler searches for symbols on Yahoo Finance
// @Summary Rechercher un symbole boursier
// @Description Recherche un symbole sur Yahoo Finance
// @Tags symbols
// @Produce json
// @Param query query string true "Terme de recherche"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/symbols/search [get]
func (h *Handler) SymbolSearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		respondError(w, http.StatusBadRequest, "INVALID_QUERY", "Query parameter is required", nil)
		return
	}

	// Call Yahoo Finance search API
	yahooService, ok := h.PriceService.(*price.YahooFinanceService)
	if !ok {
		respondError(w, http.StatusInternalServerError, "SERVICE_ERROR", "Price service is not Yahoo Finance", nil)
		return
	}

	results, err := yahooService.SearchSymbol(query)
	if err != nil {
		log.Printf("ERROR: Yahoo Finance search failed: %v", err)
		respondError(w, http.StatusBadRequest, "SEARCH_ERROR", err.Error(), nil)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

// fetchCompleteAssetPriceHistory fetches all price granularities for an asset
// This ensures we have daily data for 1M, weekly for 5Y, and max historical data
func (h *Handler) fetchCompleteAssetPriceHistory(isin string) error {
	// Get asset to retrieve symbol
	asset, err := h.DB.GetAssetByISIN(isin)
	if err != nil {
		return fmt.Errorf("asset not found: %w", err)
	}

	symbol := ""
	if asset.Symbol != nil {
		symbol = *asset.Symbol
	}

	if symbol == "" {
		return fmt.Errorf("no symbol found for asset")
	}

	// Cast to Yahoo Finance service to access FetchHistoricalPrices
	yahooService, ok := h.PriceService.(*price.YahooFinanceService)
	if !ok {
		return fmt.Errorf("price service is not Yahoo Finance")
	}

	// Fetch data in multiple periods with specific granularity
	// 1. Last month with daily data (1d interval)
	prices1m, err := yahooService.FetchHistoricalPrices(symbol, isin, asset.Currency, "1mo", "1d")
	if err != nil {
		return fmt.Errorf("failed to fetch 1m daily prices: %w", err)
	}

	// 2. 5 years with weekly data (1wk interval)
	prices5y, err := yahooService.FetchHistoricalPrices(symbol, isin, asset.Currency, "5y", "1wk")
	if err != nil {
		log.Printf("WARNING: Failed to fetch 5y weekly prices for %s: %v", isin, err)
	}

	// 3. Max range with weekly data (1wk interval)
	pricesMax, err := yahooService.FetchHistoricalPrices(symbol, isin, asset.Currency, "max", "1wk")
	if err != nil {
		log.Printf("WARNING: Failed to fetch max weekly prices for %s: %v", isin, err)
	}

	// Combine all prices and remove duplicates (keep daily over weekly for overlapping dates)
	priceMap := make(map[string]models.AssetPrice)

	// Add max prices first (lowest priority)
	for _, p := range pricesMax {
		dateKey := p.Timestamp.Format("2006-01-02")
		priceMap[dateKey] = p
	}

	// Add 5y prices (medium priority, overwrites max if same date)
	for _, p := range prices5y {
		dateKey := p.Timestamp.Format("2006-01-02")
		priceMap[dateKey] = p
	}

	// Add 1m daily prices (highest priority, overwrites weekly if same date)
	for _, p := range prices1m {
		dateKey := p.Timestamp.Format("2006-01-02")
		priceMap[dateKey] = p
	}

	// Convert map back to slice
	var allPrices []models.AssetPrice
	for _, p := range priceMap {
		allPrices = append(allPrices, p)
	}

	// Sort by timestamp
	sort.Slice(allPrices, func(i, j int) bool {
		return allPrices[i].Timestamp.Before(allPrices[j].Timestamp)
	})

	// Store all prices in database
	if len(allPrices) > 0 {
		if err := h.DB.CreateAssetPricesBatch(allPrices); err != nil {
			return fmt.Errorf("failed to store prices: %w", err)
		}
	}

	log.Printf("INFO: Stored %d price points for %s (1m daily: %d, 5y weekly: %d, max weekly: %d)",
		len(allPrices), isin, len(prices1m), len(prices5y), len(pricesMax))

	return nil
}

// resolveAssetSymbols resolves Yahoo Finance symbols for assets that don't have verified symbols
func (h *Handler) resolveAssetSymbols() int {
	yahooService, ok := h.PriceService.(*price.YahooFinanceService)
	if !ok {
		log.Printf("WARNING: Price service is not Yahoo Finance, skipping symbol resolution")
		return 0
	}

	// Get all assets without verified symbols
	query := `
		SELECT isin, name, symbol 
		FROM assets 
		WHERE (symbol_verified = false OR symbol_verified IS NULL)
		AND isin IS NOT NULL
	`

	type AssetInfo struct {
		ISIN   string  `db:"isin"`
		Name   string  `db:"name"`
		Symbol *string `db:"symbol"`
	}

	var assets []AssetInfo
	if err := h.DB.Select(&assets, query); err != nil {
		log.Printf("ERROR: Failed to get assets for symbol resolution: %v", err)
		return 0
	}

	log.Printf("INFO: Found %d assets to resolve symbols for", len(assets))

	resolved := 0
	for _, asset := range assets {
		// Get metadata from transactions to extract exchange info
		var metadata struct {
			Symbol    string   `json:"symbol"`
			Exchanges []string `json:"exchanges"`
			Name      string   `json:"name"`
		}

		// Try to get metadata from a transaction with this ISIN
		metadataQuery := `
			SELECT metadata 
			FROM transactions_traderepublic 
			WHERE isin = $1 AND metadata IS NOT NULL 
			LIMIT 1
		`
		var metadataJSON *string
		err := h.DB.Get(&metadataJSON, metadataQuery, asset.ISIN)
		if err == nil && metadataJSON != nil {
			if err := json.Unmarshal([]byte(*metadataJSON), &metadata); err != nil {
				log.Printf("WARNING: Failed to parse metadata for ISIN %s: %v", asset.ISIN, err)
			}
		}

		// Use symbol from metadata or from asset
		symbolToResolve := metadata.Symbol
		if symbolToResolve == "" && asset.Symbol != nil {
			symbolToResolve = *asset.Symbol
		}

		if symbolToResolve == "" {
			log.Printf("WARNING: No symbol found for ISIN %s, skipping", asset.ISIN)
			continue
		}

		// Use asset name from metadata or database
		assetName := metadata.Name
		if assetName == "" {
			assetName = asset.Name
		}

		// Resolve symbol with Yahoo Finance
		resolvedSymbol, verified, err := yahooService.ResolveSymbolWithExchange(
			symbolToResolve,
			metadata.Exchanges,
			assetName,
		)

		if err != nil {
			log.Printf("WARNING: Failed to resolve symbol for ISIN %s (%s): %v", asset.ISIN, symbolToResolve, err)
			continue
		}

		// Update asset with resolved symbol
		updateQuery := `
			UPDATE assets 
			SET symbol = $1, symbol_verified = $2, last_updated = NOW()
			WHERE isin = $3
		`
		if _, err := h.DB.Exec(updateQuery, resolvedSymbol, verified, asset.ISIN); err != nil {
			log.Printf("ERROR: Failed to update symbol for ISIN %s: %v", asset.ISIN, err)
			continue
		}

		log.Printf("INFO: Resolved symbol for %s: %s → %s (verified: %v)", asset.ISIN, symbolToResolve, resolvedSymbol, verified)
		resolved++

		// Fetch complete price history for this asset
		if err := h.fetchCompleteAssetPriceHistory(asset.ISIN); err != nil {
			log.Printf("WARNING: Failed to fetch price history for %s: %v", asset.ISIN, err)
		} else {
			log.Printf("INFO: Fetched complete price history for %s", asset.ISIN)
		}

		// Small delay to be respectful to Yahoo Finance
		time.Sleep(200 * time.Millisecond)
	}

	return resolved
}

// ResolveAllSymbolsHandler manually triggers symbol resolution for all assets
// @Summary Résoudre tous les symboles manquants
// @Description Déclenche la résolution des symboles Yahoo Finance pour tous les actifs
// @Tags assets
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/assets/symbols/resolve [post]
func (h *Handler) ResolveAllSymbolsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("INFO: Manual symbol resolution triggered")

	resolved := h.resolveAssetSymbols()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":          true,
		"symbols_resolved": resolved,
		"message":          fmt.Sprintf("Successfully resolved %d symbols", resolved),
	})
}

// UpdateAssetSymbolHandler updates the symbol for an asset
// @Summary Mettre à jour le symbole d'un actif
// @Description Met à jour le symbole Yahoo Finance d'un actif
// @Tags assets
// @Accept json
// @Produce json
// @Param isin path string true "Code ISIN de l'actif"
// @Param body body object true "Symbole et statut de vérification"
// @Success 200 {object} models.Asset
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/assets/{isin}/symbol [put]
func (h *Handler) UpdateAssetSymbolHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isin := vars["isin"]

	if isin == "" {
		respondError(w, http.StatusBadRequest, "INVALID_ISIN", "ISIN is required", nil)
		return
	}

	// Parse request body
	var req struct {
		Symbol         string `json:"symbol"`
		SymbolVerified bool   `json:"symbol_verified"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error())
		return
	}

	// Update asset symbol in database
	query := `
		UPDATE assets 
		SET symbol = $1, symbol_verified = $2, last_updated = NOW()
		WHERE isin = $3
		RETURNING isin, name, symbol, symbol_verified, type, currency, last_updated
	`

	var asset models.Asset
	err := h.DB.Get(&asset, query, req.Symbol, req.SymbolVerified, isin)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "ASSET_NOT_FOUND", "Asset not found", nil)
			return
		}
		log.Printf("ERROR: Failed to update asset symbol: %v", err)
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update asset symbol", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, asset)
}
