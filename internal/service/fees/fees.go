package fees

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"
)

// Service provides fee calculation and aggregation functionality
type Service interface {
	CalculateAccountFees(accountID string, startDate, endDate string) (*FeesMetrics, error)
	CalculateGlobalFees(startDate, endDate string) (*FeesMetrics, error)
}

// FeesMetrics represents aggregated fee metrics
type FeesMetrics struct {
	TotalFees        float64              `json:"total_fees"`
	AverageFees      float64              `json:"average_fees"`
	TransactionCount int                  `json:"transaction_count"`
	FeesByType       map[string]float64   `json:"fees_by_type"`
	TimeSeries       []FeeTimeSeriesPoint `json:"time_series"`
}

// FeeTimeSeriesPoint represents a point in the fee evolution chart
type FeeTimeSeriesPoint struct {
	Date string  `json:"date"`
	Fees float64 `json:"fees"`
}

// feesService implements the Service interface
type feesService struct {
	db *database.DB
}

// NewFeesService creates a new fees service
func NewFeesService(db *database.DB) Service {
	return &feesService{
		db: db,
	}
}

// CalculateAccountFees calculates fee metrics for a specific account
func (s *feesService) CalculateAccountFees(accountID string, startDate, endDate string) (*FeesMetrics, error) {
	// Get account to determine platform
	account, err := s.db.GetAccountByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Build filter
	filter := database.TransactionFilter{
		AccountID: accountID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Get all transactions for the account
	transactions, err := s.db.GetTransactionsByAccount(accountID, account.Platform, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	return s.calculateFeesFromTransactions(transactions)
}

// CalculateGlobalFees calculates fee metrics across all accounts
func (s *feesService) CalculateGlobalFees(startDate, endDate string) (*FeesMetrics, error) {
	// Get all accounts
	accounts, err := s.db.GetAllAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	// Collect all transactions from all accounts
	allTransactions := []models.Transaction{}

	for _, account := range accounts {
		filter := database.TransactionFilter{
			AccountID: account.ID,
			StartDate: startDate,
			EndDate:   endDate,
		}

		transactions, err := s.db.GetTransactionsByAccount(account.ID, account.Platform, filter)
		if err != nil {
			// Log error but continue with other accounts
			continue
		}

		allTransactions = append(allTransactions, transactions...)
	}

	return s.calculateFeesFromTransactions(allTransactions)
}

// calculateFeesFromTransactions calculates fee metrics from a list of transactions
func (s *feesService) calculateFeesFromTransactions(transactions []models.Transaction) (*FeesMetrics, error) {
	metrics := &FeesMetrics{
		TotalFees:        0,
		AverageFees:      0,
		TransactionCount: 0,
		FeesByType:       make(map[string]float64),
		TimeSeries:       []FeeTimeSeriesPoint{},
	}

	if len(transactions) == 0 {
		return metrics, nil
	}

	// Map to aggregate fees by date for time series
	feesByDate := make(map[string]float64)

	// Process each transaction
	for _, tx := range transactions {
		// Parse fees from the Fees field (format: "X,XX €" or "X.XX €")
		feeValue := parseFeeValue(tx.Fees)

		if feeValue > 0 {
			metrics.TotalFees += feeValue
			metrics.TransactionCount++

			// Aggregate by transaction type
			txType := tx.TransactionType
			if txType == "" {
				txType = "unknown"
			}
			metrics.FeesByType[txType] += feeValue

			// Aggregate by date for time series
			date := extractDate(tx.Timestamp)
			if date != "" {
				feesByDate[date] += feeValue
			}
		}
	}

	// Calculate average fees
	if metrics.TransactionCount > 0 {
		metrics.AverageFees = metrics.TotalFees / float64(metrics.TransactionCount)
	}

	// Build time series from aggregated data
	for date, fees := range feesByDate {
		metrics.TimeSeries = append(metrics.TimeSeries, FeeTimeSeriesPoint{
			Date: date,
			Fees: fees,
		})
	}

	// Sort time series by date
	sortTimeSeries(metrics.TimeSeries)

	return metrics, nil
}

// parseFeeValue parses a fee string (e.g., "1,00 €" or "1.50 €") to a float64
func parseFeeValue(feeStr string) float64 {
	if feeStr == "" {
		return 0
	}

	// Remove currency symbols and whitespace
	feeStr = strings.TrimSpace(feeStr)
	feeStr = strings.ReplaceAll(feeStr, "€", "")
	feeStr = strings.ReplaceAll(feeStr, "$", "")
	feeStr = strings.ReplaceAll(feeStr, "USD", "")
	feeStr = strings.ReplaceAll(feeStr, "EUR", "")
	feeStr = strings.TrimSpace(feeStr)

	// Replace comma with dot for parsing
	feeStr = strings.ReplaceAll(feeStr, ",", ".")

	// Parse to float
	value, err := strconv.ParseFloat(feeStr, 64)
	if err != nil {
		return 0
	}

	// Return absolute value (fees should be positive)
	if value < 0 {
		return -value
	}

	return value
}

// extractDate extracts the date part (YYYY-MM-DD) from a timestamp
func extractDate(timestamp string) string {
	if timestamp == "" {
		return ""
	}

	// Try to parse as RFC3339
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// Try to extract just the date part if it's already in YYYY-MM-DD format
		if len(timestamp) >= 10 {
			return timestamp[:10]
		}
		return ""
	}

	return t.Format("2006-01-02")
}

// sortTimeSeries sorts time series points by date in ascending order
func sortTimeSeries(series []FeeTimeSeriesPoint) {
	// Simple bubble sort for small datasets
	n := len(series)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if series[j].Date > series[j+1].Date {
				series[j], series[j+1] = series[j+1], series[j]
			}
		}
	}
}
