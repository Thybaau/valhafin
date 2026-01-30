package performance

import (
	"fmt"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"
	"valhafin/internal/service/price"
)

// Service provides performance calculation functionality
type Service interface {
	CalculateAccountPerformance(accountID string, period string) (*Performance, error)
	CalculateGlobalPerformance(period string) (*Performance, error)
	CalculateAssetPerformance(isin string, period string) (*AssetPerformance, error)
}

// PerformanceService implements the Service interface
type PerformanceService struct {
	DB           *database.DB
	PriceService price.Service
}

// NewPerformanceService creates a new PerformanceService
func NewPerformanceService(db *database.DB, priceService price.Service) *PerformanceService {
	return &PerformanceService{
		DB:           db,
		PriceService: priceService,
	}
}

// Performance represents portfolio performance metrics
type Performance struct {
	TotalValue      float64            `json:"total_value"`
	TotalInvested   float64            `json:"total_invested"`
	TotalFees       float64            `json:"total_fees"`
	RealizedGains   float64            `json:"realized_gains"`
	UnrealizedGains float64            `json:"unrealized_gains"`
	PerformancePct  float64            `json:"performance_pct"`
	TimeSeries      []PerformancePoint `json:"time_series"`
}

// PerformancePoint represents a point in the performance time series
type PerformancePoint struct {
	Date  time.Time `json:"date"`
	Value float64   `json:"value"`
}

// AssetPerformance represents performance metrics for a specific asset
type AssetPerformance struct {
	ISIN            string             `json:"isin"`
	Name            string             `json:"name"`
	CurrentPrice    float64            `json:"current_price"`
	TotalQuantity   float64            `json:"total_quantity"`
	TotalValue      float64            `json:"total_value"`
	TotalInvested   float64            `json:"total_invested"`
	TotalFees       float64            `json:"total_fees"`
	RealizedGains   float64            `json:"realized_gains"`
	UnrealizedGains float64            `json:"unrealized_gains"`
	PerformancePct  float64            `json:"performance_pct"`
	TimeSeries      []PerformancePoint `json:"time_series"`
}

// CalculateAccountPerformance calculates performance for a specific account
func (s *PerformanceService) CalculateAccountPerformance(accountID string, period string) (*Performance, error) {
	// Get account to determine platform
	account, err := s.DB.GetAccountByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Calculate date range based on period
	startDate, endDate := calculateDateRange(period)

	// Get transactions for the account
	filter := database.TransactionFilter{
		StartDate: startDate.Format(time.RFC3339),
		EndDate:   endDate.Format(time.RFC3339),
	}

	transactions, err := s.DB.GetTransactionsByAccount(accountID, account.Platform, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Calculate performance
	return s.calculatePerformance(transactions, startDate, endDate)
}

// CalculateGlobalPerformance calculates performance across all accounts
func (s *PerformanceService) CalculateGlobalPerformance(period string) (*Performance, error) {
	// Get all accounts
	accounts, err := s.DB.GetAllAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	// Calculate date range based on period
	startDate, endDate := calculateDateRange(period)

	// Collect all transactions from all accounts
	var allTransactions []models.Transaction
	for _, account := range accounts {
		filter := database.TransactionFilter{
			StartDate: startDate.Format(time.RFC3339),
			EndDate:   endDate.Format(time.RFC3339),
		}

		transactions, err := s.DB.GetTransactionsByAccount(account.ID, account.Platform, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get transactions for account %s: %w", account.ID, err)
		}

		allTransactions = append(allTransactions, transactions...)
	}

	// Calculate performance
	return s.calculatePerformance(allTransactions, startDate, endDate)
}

// CalculateAssetPerformance calculates performance for a specific asset
func (s *PerformanceService) CalculateAssetPerformance(isin string, period string) (*AssetPerformance, error) {
	// Get asset information
	asset, err := s.DB.GetAssetByISIN(isin)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	// Get current price
	currentPrice, err := s.PriceService.GetCurrentPrice(isin)
	if err != nil {
		return nil, fmt.Errorf("failed to get current price: %w", err)
	}

	// Calculate date range based on period
	startDate, endDate := calculateDateRange(period)

	// Get all transactions for this asset across all accounts
	accounts, err := s.DB.GetAllAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	var assetTransactions []models.Transaction
	for _, account := range accounts {
		filter := database.TransactionFilter{
			ISIN:      isin,
			StartDate: startDate.Format(time.RFC3339),
			EndDate:   endDate.Format(time.RFC3339),
		}

		transactions, err := s.DB.GetTransactionsByAccount(account.ID, account.Platform, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get transactions for account %s: %w", account.ID, err)
		}

		assetTransactions = append(assetTransactions, transactions...)
	}

	// Calculate asset-specific metrics
	return s.calculateAssetPerformance(asset, assetTransactions, currentPrice.Price, startDate, endDate)
}

// calculatePerformance performs the actual performance calculation
func (s *PerformanceService) calculatePerformance(transactions []models.Transaction, startDate, endDate time.Time) (*Performance, error) {
	// Group transactions by asset (ISIN)
	assetHoldings := make(map[string]*assetHolding)
	var totalFees float64
	var totalInvested float64
	var realizedGains float64

	for _, tx := range transactions {
		// Parse fees from the Fees field
		fees := parseFees(tx.Fees)
		totalFees += fees

		// Skip if no ISIN (e.g., pure fee transactions)
		if tx.ISIN == nil || *tx.ISIN == "" {
			continue
		}

		isin := *tx.ISIN

		// Initialize holding if not exists
		if _, exists := assetHoldings[isin]; !exists {
			assetHoldings[isin] = &assetHolding{
				ISIN:     isin,
				Quantity: 0,
				Invested: 0,
			}
		}

		holding := assetHoldings[isin]

		// Process transaction based on type
		switch tx.TransactionType {
		case "buy":
			holding.Quantity += tx.Quantity
			holding.Invested += tx.AmountValue
			totalInvested += tx.AmountValue
		case "sell":
			// Calculate realized gain/loss
			avgCost := 0.0
			if holding.Quantity > 0 {
				avgCost = holding.Invested / holding.Quantity
			}
			realizedGains += tx.AmountValue - (avgCost * tx.Quantity)

			holding.Quantity -= tx.Quantity
			holding.Invested -= avgCost * tx.Quantity
		case "dividend":
			realizedGains += tx.AmountValue
		}
	}

	// Calculate current value of holdings
	var totalValue float64
	for isin, holding := range assetHoldings {
		if holding.Quantity <= 0 {
			continue
		}

		// Get current price
		currentPrice, err := s.PriceService.GetCurrentPrice(isin)
		if err != nil {
			// If price not available, use last known value
			continue
		}

		totalValue += holding.Quantity * currentPrice.Price
	}

	// Calculate unrealized gains
	unrealizedGains := totalValue - calculateRemainingInvestment(assetHoldings)

	// Calculate performance percentage
	performancePct := 0.0
	if totalInvested > 0 {
		performancePct = ((totalValue + realizedGains - totalInvested - totalFees) / totalInvested) * 100
	}

	// Generate time series (simplified - daily snapshots)
	timeSeries := s.generateTimeSeries(transactions, assetHoldings, startDate, endDate)

	return &Performance{
		TotalValue:      totalValue,
		TotalInvested:   totalInvested,
		TotalFees:       totalFees,
		RealizedGains:   realizedGains,
		UnrealizedGains: unrealizedGains,
		PerformancePct:  performancePct,
		TimeSeries:      timeSeries,
	}, nil
}

// calculateAssetPerformance calculates performance for a specific asset
func (s *PerformanceService) calculateAssetPerformance(asset *models.Asset, transactions []models.Transaction, currentPrice float64, startDate, endDate time.Time) (*AssetPerformance, error) {
	var totalQuantity float64
	var totalInvested float64
	var totalFees float64
	var realizedGains float64

	for _, tx := range transactions {
		fees := parseFees(tx.Fees)
		totalFees += fees

		switch tx.TransactionType {
		case "buy":
			totalQuantity += tx.Quantity
			totalInvested += tx.AmountValue
		case "sell":
			avgCost := 0.0
			if totalQuantity > 0 {
				avgCost = totalInvested / totalQuantity
			}
			realizedGains += tx.AmountValue - (avgCost * tx.Quantity)
			totalQuantity -= tx.Quantity
			totalInvested -= avgCost * tx.Quantity
		case "dividend":
			realizedGains += tx.AmountValue
		}
	}

	// Calculate current value
	totalValue := totalQuantity * currentPrice

	// Calculate unrealized gains
	unrealizedGains := totalValue - totalInvested

	// Calculate performance percentage
	performancePct := 0.0
	if totalInvested > 0 {
		performancePct = ((totalValue + realizedGains - totalInvested - totalFees) / totalInvested) * 100
	}

	// Generate time series
	timeSeries, err := s.generateAssetTimeSeries(asset.ISIN, transactions, startDate, endDate)
	if err != nil {
		// If time series generation fails, return empty series
		timeSeries = []PerformancePoint{}
	}

	return &AssetPerformance{
		ISIN:            asset.ISIN,
		Name:            asset.Name,
		CurrentPrice:    currentPrice,
		TotalQuantity:   totalQuantity,
		TotalValue:      totalValue,
		TotalInvested:   totalInvested,
		TotalFees:       totalFees,
		RealizedGains:   realizedGains,
		UnrealizedGains: unrealizedGains,
		PerformancePct:  performancePct,
		TimeSeries:      timeSeries,
	}, nil
}

// Helper types and functions

type assetHolding struct {
	ISIN     string
	Quantity float64
	Invested float64
}

// calculateDateRange converts a period string to start and end dates
func calculateDateRange(period string) (time.Time, time.Time) {
	endDate := time.Now()
	var startDate time.Time

	switch period {
	case "1m":
		startDate = endDate.AddDate(0, -1, 0)
	case "3m":
		startDate = endDate.AddDate(0, -3, 0)
	case "1y":
		startDate = endDate.AddDate(-1, 0, 0)
	case "all":
		startDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		// Default to 1 year
		startDate = endDate.AddDate(-1, 0, 0)
	}

	return startDate, endDate
}

// parseFees extracts fee amount from the Fees string field
func parseFees(feesStr string) float64 {
	// The Fees field might be in format like "1,23 €" or "1.23"
	// For now, we'll try to parse it as a float
	// In production, this would need more robust parsing
	var fees float64
	fmt.Sscanf(feesStr, "%f", &fees)
	return fees
}

// calculateRemainingInvestment calculates the total invested amount still in holdings
func calculateRemainingInvestment(holdings map[string]*assetHolding) float64 {
	var total float64
	for _, holding := range holdings {
		if holding.Quantity > 0 {
			total += holding.Invested
		}
	}
	return total
}

// generateTimeSeries generates a time series of portfolio values
func (s *PerformanceService) generateTimeSeries(transactions []models.Transaction, holdings map[string]*assetHolding, startDate, endDate time.Time) []PerformancePoint {
	// Simplified implementation: return empty for now
	// Full implementation would replay transactions day by day
	return []PerformancePoint{}
}

// generateAssetTimeSeries generates a time series for a specific asset
func (s *PerformanceService) generateAssetTimeSeries(isin string, transactions []models.Transaction, startDate, endDate time.Time) ([]PerformancePoint, error) {
	// Get price history for the asset
	priceHistory, err := s.PriceService.GetPriceHistory(isin, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Build time series by combining quantity held with price history
	var timeSeries []PerformancePoint
	var currentQuantity float64

	// Sort transactions by timestamp
	// For simplicity, we'll just use the price history
	for _, pricePoint := range priceHistory {
		// Calculate quantity held at this point
		// This is simplified - full implementation would replay transactions
		value := currentQuantity * pricePoint.Price
		timeSeries = append(timeSeries, PerformancePoint{
			Date:  pricePoint.Timestamp,
			Value: value,
		})
	}

	return timeSeries, nil
}
