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
	CashBalance     float64            `json:"cash_balance"`
	TotalFees       float64            `json:"total_fees"`
	RealizedGains   float64            `json:"realized_gains"`
	UnrealizedGains float64            `json:"unrealized_gains"`
	PerformancePct  float64            `json:"performance_pct"`
	TimeSeries      []PerformancePoint `json:"time_series"`
}

// PerformancePoint represents a point in the performance time series
type PerformancePoint struct {
	Date     time.Time `json:"date"`
	Value    float64   `json:"value"`    // Current value of assets in EUR
	Invested float64   `json:"invested"` // Amount invested in assets (cost basis)
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

	// Collect filtered transactions (for period-specific metrics)
	var filteredTransactions []models.Transaction
	for _, account := range accounts {
		filter := database.TransactionFilter{
			StartDate: startDate.Format(time.RFC3339),
			EndDate:   endDate.Format(time.RFC3339),
		}

		transactions, err := s.DB.GetTransactionsByAccount(account.ID, account.Platform, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get transactions for account %s: %w", account.ID, err)
		}

		filteredTransactions = append(filteredTransactions, transactions...)
	}

	// Collect ALL transactions (for cash balance calculation only)
	var allTransactions []models.Transaction
	for _, account := range accounts {
		filter := database.TransactionFilter{
			Limit: 10000, // Get all transactions
		}

		transactions, err := s.DB.GetTransactionsByAccount(account.ID, account.Platform, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get transactions for account %s: %w", account.ID, err)
		}

		allTransactions = append(allTransactions, transactions...)
	}

	// Calculate performance with filtered transactions
	performance, err := s.calculatePerformance(filteredTransactions, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Recalculate cash balance using ALL transactions
	cashBalance := s.calculateCashBalance(allTransactions)
	performance.CashBalance = cashBalance

	return performance, nil
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
	var totalInvested float64 // Total amount invested (all buys, including sold positions)
	var totalDeposits float64
	var totalInterests float64
	var totalSales float64 // Total amount from sales

	for _, tx := range transactions {
		// Parse fees from the Fees field
		fees := parseFees(tx.Fees)
		totalFees += fees

		// Handle different transaction types
		switch tx.TransactionType {
		case "deposit":
			totalDeposits += tx.AmountValue
			continue
		case "withdrawal":
			totalDeposits += tx.AmountValue // AmountValue is negative for withdrawals
			continue
		case "interest":
			totalInterests += tx.AmountValue
			continue
		case "fee":
			continue
		case "dividend":
			// Dividends are added to interests
			totalInterests += tx.AmountValue
			continue
		}

		// Skip if no ISIN
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
			// AmountValue represents the cost of the purchase (positive value)
			investedAmount := tx.AmountValue
			if investedAmount < 0 {
				investedAmount = -investedAmount // Handle negative values if they exist
			}
			holding.Invested += investedAmount
			// Add to total invested (all buys, even if later sold)
			totalInvested += investedAmount
		case "sell":
			// Track total sales amount (positive value)
			saleAmount := tx.AmountValue
			if saleAmount < 0 {
				saleAmount = -saleAmount // Handle negative values if they exist
			}
			totalSales += saleAmount
			// Calculate realized gain/loss
			avgCost := 0.0
			if holding.Quantity > 0 {
				avgCost = holding.Invested / holding.Quantity
			}
			holding.Quantity -= tx.Quantity
			holding.Invested -= avgCost * tx.Quantity
		}
	}

	// Calculate current value of holdings (assets only, no cash)
	var assetsValue float64
	var currentInvested float64 // Amount currently invested (still in holdings)
	for isin, holding := range assetHoldings {
		if holding.Quantity <= 0 {
			continue
		}

		// Add to current invested amount
		currentInvested += holding.Invested

		// Get current price
		currentPrice, err := s.PriceService.GetCurrentPrice(isin)
		if err != nil {
			// If price not available, use invested value as fallback
			assetsValue += holding.Invested
			continue
		}

		assetsValue += holding.Quantity * currentPrice.Price
	}

	// Calculate cash balance: deposits - buys + sells + interests - fees
	// This represents the actual cash remaining in the account
	cashBalance := totalDeposits - totalInvested + totalSales + totalInterests - totalFees

	// Total value = current value of assets only (no cash)
	totalValue := assetsValue

	// Calculate unrealized gains (current value of assets - invested amount still in holdings)
	unrealizedGains := assetsValue - currentInvested

	// Calculate performance percentage based on current investment
	// Formula: performance % = ((current_value - total_invested - total_fees) / total_invested) × 100
	performancePct := 0.0
	if currentInvested > 0 {
		performancePct = ((assetsValue - currentInvested - totalFees) / currentInvested) * 100
	}

	// Generate time series
	timeSeries := s.generateTimeSeries(transactions, assetHoldings, startDate, endDate)

	return &Performance{
		TotalValue:      totalValue,
		TotalInvested:   currentInvested, // Amount currently invested in open positions
		CashBalance:     cashBalance,
		TotalFees:       totalFees,
		RealizedGains:   totalSales + totalInterests - totalFees, // Realized gains from sales + interests - fees
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

// calculateCashBalance calculates the current cash balance using all transactions
// Cash = deposits - buys + sells + interests - fees
func (s *PerformanceService) calculateCashBalance(transactions []models.Transaction) float64 {
	var totalDeposits float64
	var totalInvested float64 // Total amount spent on buys
	var totalSales float64    // Total amount received from sells
	var totalInterests float64
	var totalFees float64

	for _, tx := range transactions {
		// Parse fees
		fees := parseFees(tx.Fees)
		totalFees += fees

		// Handle different transaction types
		switch tx.TransactionType {
		case "deposit":
			totalDeposits += tx.AmountValue
		case "withdrawal":
			totalDeposits += tx.AmountValue // AmountValue is negative for withdrawals
		case "interest":
			totalInterests += tx.AmountValue
		case "dividend":
			totalInterests += tx.AmountValue
		case "buy":
			if tx.ISIN != nil && *tx.ISIN != "" {
				investedAmount := tx.AmountValue
				if investedAmount < 0 {
					investedAmount = -investedAmount
				}
				totalInvested += investedAmount
			}
		case "sell":
			if tx.ISIN != nil && *tx.ISIN != "" {
				saleAmount := tx.AmountValue
				if saleAmount < 0 {
					saleAmount = -saleAmount
				}
				totalSales += saleAmount
			}
		}
	}

	// Cash balance = deposits - buys + sells + interests - fees
	return totalDeposits - totalInvested + totalSales + totalInterests - totalFees
}

// generateTimeSeries generates a time series of portfolio values using historical prices
// This creates a Trade Republic-style performance chart with weekly data points
func (s *PerformanceService) generateTimeSeries(transactions []models.Transaction, holdings map[string]*assetHolding, startDate, endDate time.Time) []PerformancePoint {
	if len(transactions) == 0 {
		return []PerformancePoint{}
	}

	// Sort transactions by timestamp
	sortedTxs := make([]models.Transaction, len(transactions))
	copy(sortedTxs, transactions)

	// Simple bubble sort by timestamp
	for i := 0; i < len(sortedTxs); i++ {
		for j := i + 1; j < len(sortedTxs); j++ {
			ti, _ := time.Parse(time.RFC3339, sortedTxs[i].Timestamp)
			tj, _ := time.Parse(time.RFC3339, sortedTxs[j].Timestamp)
			if ti.After(tj) {
				sortedTxs[i], sortedTxs[j] = sortedTxs[j], sortedTxs[i]
			}
		}
	}

	// Determine the actual start date (first transaction date)
	firstTxTime, err := time.Parse(time.RFC3339, sortedTxs[0].Timestamp)
	if err != nil {
		firstTxTime = startDate
	}

	// Use the later of startDate or first transaction date
	if firstTxTime.After(startDate) {
		startDate = firstTxTime
	}

	// Generate weekly time points from start to end
	var timePoints []time.Time
	currentPoint := startDate

	// Determine interval based on period length
	daysDiff := endDate.Sub(startDate).Hours() / 24
	var interval time.Duration

	if daysDiff <= 7 {
		interval = 24 * time.Hour // Daily for 1 week
	} else if daysDiff <= 30 {
		interval = 24 * time.Hour // Daily for 1 month
	} else if daysDiff <= 90 {
		interval = 3 * 24 * time.Hour // Every 3 days for 3 months
	} else {
		interval = 7 * 24 * time.Hour // Weekly for longer periods
	}

	for currentPoint.Before(endDate) || currentPoint.Equal(endDate) {
		timePoints = append(timePoints, currentPoint)
		currentPoint = currentPoint.Add(interval)
	}

	// Always add the end date as the last point
	if len(timePoints) == 0 || !timePoints[len(timePoints)-1].Equal(endDate) {
		timePoints = append(timePoints, endDate)
	}

	// Build time series by replaying transactions and using historical prices
	var timeSeries []PerformancePoint
	currentHoldings := make(map[string]*assetHolding)
	txIndex := 0

	for _, timePoint := range timePoints {
		// Process all transactions up to this time point
		for txIndex < len(sortedTxs) {
			txTime, err := time.Parse(time.RFC3339, sortedTxs[txIndex].Timestamp)
			if err != nil || txTime.After(timePoint) {
				break
			}

			tx := sortedTxs[txIndex]

			// Process transaction
			switch tx.TransactionType {
			case "buy":
				if tx.ISIN != nil && *tx.ISIN != "" {
					isin := *tx.ISIN
					if _, exists := currentHoldings[isin]; !exists {
						currentHoldings[isin] = &assetHolding{ISIN: isin, Quantity: 0, Invested: 0}
					}
					currentHoldings[isin].Quantity += tx.Quantity
					// Track cost basis
					investedAmount := tx.AmountValue
					if investedAmount < 0 {
						investedAmount = -investedAmount
					}
					currentHoldings[isin].Invested += investedAmount
				}
			case "sell":
				if tx.ISIN != nil && *tx.ISIN != "" {
					isin := *tx.ISIN
					if holding, exists := currentHoldings[isin]; exists {
						// Reduce cost basis proportionally
						avgCost := 0.0
						if holding.Quantity > 0 {
							avgCost = holding.Invested / holding.Quantity
						}
						holding.Quantity -= tx.Quantity
						holding.Invested -= avgCost * tx.Quantity
					}
				}
			}

			txIndex++
		}

		// Calculate portfolio value and total invested at this time point
		portfolioValue := 0.0
		totalInvested := 0.0

		for isin, holding := range currentHoldings {
			if holding.Quantity <= 0 {
				continue
			}

			// Add to total invested (cost basis)
			totalInvested += holding.Invested

			// Get historical price for this date
			price, err := s.getHistoricalPrice(isin, timePoint)
			if err != nil {
				// Skip if no price available
				continue
			}

			portfolioValue += holding.Quantity * price
		}

		// Add point to time series
		timeSeries = append(timeSeries, PerformancePoint{
			Date:     timePoint,
			Value:    portfolioValue,
			Invested: totalInvested,
		})
	}

	return timeSeries
}

// getHistoricalPrice retrieves the historical price for an asset at a specific date
// It looks for the closest price in the database
func (s *PerformanceService) getHistoricalPrice(isin string, date time.Time) (float64, error) {
	// Check if DB is available (for tests)
	if s.DB == nil {
		// Fallback to current price service
		currentPrice, err := s.PriceService.GetCurrentPrice(isin)
		if err != nil {
			return 0, fmt.Errorf("no price available for %s at %s", isin, date.Format("2006-01-02"))
		}
		return currentPrice.Price, nil
	}

	// Query for the closest price to the given date
	query := `
		SELECT price 
		FROM asset_prices 
		WHERE isin = $1 
		AND timestamp <= $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var price float64
	err := s.DB.Get(&price, query, isin, date)
	if err != nil {
		// If no historical price found, try to get current price as fallback
		currentPrice, err := s.PriceService.GetCurrentPrice(isin)
		if err != nil {
			return 0, fmt.Errorf("no price available for %s at %s", isin, date.Format("2006-01-02"))
		}
		return currentPrice.Price, nil
	}

	return price, nil
}

// generateAssetTimeSeries generates a time series for a specific asset
// This replays transactions and uses historical prices to show asset value evolution
func (s *PerformanceService) generateAssetTimeSeries(isin string, transactions []models.Transaction, startDate, endDate time.Time) ([]PerformancePoint, error) {
	if len(transactions) == 0 {
		return []PerformancePoint{}, nil
	}

	// Sort transactions by timestamp
	sortedTxs := make([]models.Transaction, len(transactions))
	copy(sortedTxs, transactions)

	for i := 0; i < len(sortedTxs); i++ {
		for j := i + 1; j < len(sortedTxs); j++ {
			ti, _ := time.Parse(time.RFC3339, sortedTxs[i].Timestamp)
			tj, _ := time.Parse(time.RFC3339, sortedTxs[j].Timestamp)
			if ti.After(tj) {
				sortedTxs[i], sortedTxs[j] = sortedTxs[j], sortedTxs[i]
			}
		}
	}

	// Determine the actual start date (first transaction date)
	firstTxTime, err := time.Parse(time.RFC3339, sortedTxs[0].Timestamp)
	if err != nil {
		firstTxTime = startDate
	}

	if firstTxTime.After(startDate) {
		startDate = firstTxTime
	}

	// Generate time points
	var timePoints []time.Time
	currentPoint := startDate

	daysDiff := endDate.Sub(startDate).Hours() / 24
	var interval time.Duration

	if daysDiff <= 7 {
		interval = 24 * time.Hour
	} else if daysDiff <= 30 {
		interval = 24 * time.Hour
	} else if daysDiff <= 90 {
		interval = 3 * 24 * time.Hour
	} else {
		interval = 7 * 24 * time.Hour
	}

	for currentPoint.Before(endDate) || currentPoint.Equal(endDate) {
		timePoints = append(timePoints, currentPoint)
		currentPoint = currentPoint.Add(interval)
	}

	if len(timePoints) == 0 || !timePoints[len(timePoints)-1].Equal(endDate) {
		timePoints = append(timePoints, endDate)
	}

	// Build time series
	var timeSeries []PerformancePoint
	var currentQuantity float64
	var totalInvested float64
	txIndex := 0

	for _, timePoint := range timePoints {
		// Process all transactions up to this time point
		for txIndex < len(sortedTxs) {
			txTime, err := time.Parse(time.RFC3339, sortedTxs[txIndex].Timestamp)
			if err != nil || txTime.After(timePoint) {
				break
			}

			tx := sortedTxs[txIndex]

			switch tx.TransactionType {
			case "buy":
				currentQuantity += tx.Quantity
				investedAmount := tx.AmountValue
				if investedAmount < 0 {
					investedAmount = -investedAmount
				}
				totalInvested += investedAmount
			case "sell":
				// Reduce cost basis proportionally
				avgCost := 0.0
				if currentQuantity > 0 {
					avgCost = totalInvested / currentQuantity
				}
				currentQuantity -= tx.Quantity
				totalInvested -= avgCost * tx.Quantity
			}

			txIndex++
		}

		// Get historical price for this date
		price, err := s.getHistoricalPrice(isin, timePoint)
		if err != nil {
			continue
		}

		// Calculate value
		value := currentQuantity * price

		timeSeries = append(timeSeries, PerformancePoint{
			Date:     timePoint,
			Value:    value,
			Invested: totalInvested,
		})
	}

	return timeSeries, nil
}
