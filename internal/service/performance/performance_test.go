package performance

import (
	"testing"
	"time"
	"valhafin/internal/domain/models"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// MockPriceService is a mock implementation of price.Service for testing
type MockPriceService struct {
	prices map[string]float64
}

func NewMockPriceService() *MockPriceService {
	return &MockPriceService{
		prices: make(map[string]float64),
	}
}

func (m *MockPriceService) GetCurrentPrice(isin string) (*models.AssetPrice, error) {
	price, exists := m.prices[isin]
	if !exists {
		price = 100.0 // Default price
	}
	return &models.AssetPrice{
		ISIN:      isin,
		Price:     price,
		Currency:  "EUR",
		Timestamp: time.Now(),
	}, nil
}

func (m *MockPriceService) GetPriceHistory(isin string, startDate, endDate time.Time) ([]models.AssetPrice, error) {
	// Return a simple price history
	var history []models.AssetPrice
	currentDate := startDate
	price := 100.0

	for currentDate.Before(endDate) {
		history = append(history, models.AssetPrice{
			ISIN:      isin,
			Price:     price,
			Currency:  "EUR",
			Timestamp: currentDate,
		})
		currentDate = currentDate.AddDate(0, 0, 1)
		price += 1.0 // Increment price by 1 each day
	}

	return history, nil
}

func (m *MockPriceService) UpdateAllPrices() error {
	return nil
}

func (m *MockPriceService) UpdateAssetPrice(isin string) error {
	return nil
}

func (m *MockPriceService) SetPrice(isin string, price float64) {
	m.prices[isin] = price
}

// TestCalculateDateRange tests the date range calculation
func TestCalculateDateRange(t *testing.T) {
	tests := []struct {
		name   string
		period string
	}{
		{"1 month", "1m"},
		{"3 months", "3m"},
		{"1 year", "1y"},
		{"all time", "all"},
		{"default", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startDate, endDate := calculateDateRange(tt.period)
			if startDate.After(endDate) {
				t.Errorf("startDate %v is after endDate %v", startDate, endDate)
			}
		})
	}
}

// TestParseFees tests the fee parsing function
func TestParseFees(t *testing.T) {
	tests := []struct {
		name     string
		feesStr  string
		expected float64
	}{
		{"simple float", "1.23", 1.23},
		{"zero", "0", 0},
		{"empty", "", 0},
		{"with currency", "1.23 €", 1.23},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFees(tt.feesStr)
			if result != tt.expected {
				t.Errorf("parseFees(%q) = %v, want %v", tt.feesStr, result, tt.expected)
			}
		})
	}
}

// **Propriété 10: Calcul de performance avec prix actuels**
// **Valide: Exigences 4.4, 4.6, 10.7**
//
// Property: For all portfolios, the calculated performance must use current asset prices
// (retrieved via ISIN from financial API) and include all transaction fees in the calculation.
// The formula must be: performance % = ((current_value - total_invested - total_fees) / total_invested) × 100
func TestProperty_PerformanceCalculationWithCurrentPrices(t *testing.T) {
	mockPriceService := NewMockPriceService()
	service := &PerformanceService{
		DB:           nil, // Not needed for this test
		PriceService: mockPriceService,
	}

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("performance calculation uses current prices and includes fees", prop.ForAll(
		func(quantity float64, buyPrice float64, currentPrice float64, fees float64) bool {
			// Ensure positive values
			if quantity <= 0 || buyPrice <= 0 || currentPrice <= 0 || fees < 0 {
				return true // Skip invalid inputs
			}

			// Create a simple transaction set: one buy
			isin := "TEST123456"
			mockPriceService.SetPrice(isin, currentPrice)

			transactions := []models.Transaction{
				{
					ID:              "tx1",
					AccountID:       "acc1",
					ISIN:            stringPtr(isin),
					Quantity:        quantity,
					AmountValue:     quantity * buyPrice,
					TransactionType: "buy",
					Fees:            formatFees(fees),
					Timestamp:       time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				},
			}

			// Calculate performance
			startDate := time.Now().AddDate(0, 0, -7)
			endDate := time.Now()
			performance, err := service.calculatePerformance(transactions, startDate, endDate)
			if err != nil {
				t.Logf("calculatePerformance failed: %v", err)
				return false
			}

			// Verify calculations
			expectedTotalValue := quantity * currentPrice
			expectedTotalInvested := quantity * buyPrice
			expectedTotalFees := fees

			// Check total value (with small tolerance for floating point)
			if !floatEquals(performance.TotalValue, expectedTotalValue, 0.01) {
				t.Logf("TotalValue mismatch: got %v, want %v", performance.TotalValue, expectedTotalValue)
				return false
			}

			// Check total invested
			if !floatEquals(performance.TotalInvested, expectedTotalInvested, 0.01) {
				t.Logf("TotalInvested mismatch: got %v, want %v", performance.TotalInvested, expectedTotalInvested)
				return false
			}

			// Check total fees
			if !floatEquals(performance.TotalFees, expectedTotalFees, 0.01) {
				t.Logf("TotalFees mismatch: got %v, want %v", performance.TotalFees, expectedTotalFees)
				return false
			}

			// Verify performance percentage formula
			expectedPerformancePct := ((expectedTotalValue - expectedTotalInvested - expectedTotalFees) / expectedTotalInvested) * 100
			if !floatEquals(performance.PerformancePct, expectedPerformancePct, 0.01) {
				t.Logf("PerformancePct mismatch: got %v, want %v", performance.PerformancePct, expectedPerformancePct)
				return false
			}

			return true
		},
		gen.Float64Range(1, 100),   // quantity
		gen.Float64Range(10, 1000), // buy price
		gen.Float64Range(10, 1000), // current price
		gen.Float64Range(0, 50),    // fees
	))

	properties.Property("performance includes fees in calculation", prop.ForAll(
		func(quantity float64, price float64, fees float64) bool {
			if quantity <= 0 || price <= 0 || fees < 0 {
				return true
			}

			isin := "TEST123456"
			mockPriceService.SetPrice(isin, price)

			transactions := []models.Transaction{
				{
					ID:              "tx1",
					AccountID:       "acc1",
					ISIN:            stringPtr(isin),
					Quantity:        quantity,
					AmountValue:     quantity * price,
					TransactionType: "buy",
					Fees:            formatFees(fees),
					Timestamp:       time.Now().Format(time.RFC3339),
				},
			}

			startDate := time.Now().AddDate(0, 0, -1)
			endDate := time.Now()
			performance, err := service.calculatePerformance(transactions, startDate, endDate)
			if err != nil {
				return false
			}

			// Fees must be included in the calculation
			if !floatEquals(performance.TotalFees, fees, 0.01) {
				t.Logf("Fees not properly included: got %v, want %v", performance.TotalFees, fees)
				return false
			}

			// Performance should account for fees
			// With same buy and current price, performance should be negative due to fees
			if fees > 0 && performance.PerformancePct >= 0 {
				t.Logf("Performance should be negative when fees > 0 and price unchanged")
				return false
			}

			return true
		},
		gen.Float64Range(1, 100),
		gen.Float64Range(10, 1000),
		gen.Float64Range(1, 50),
	))

	properties.TestingRun(t)
}

// **Propriété 11: Agrégation de performance globale**
// **Valide: Exigences 4.2**
//
// Property: For all sets of accounts, the global performance must be the sum of
// individual account performances and must reflect the total value of all held assets.
func TestProperty_GlobalPerformanceAggregation(t *testing.T) {
	mockPriceService := NewMockPriceService()
	service := &PerformanceService{
		DB:           nil,
		PriceService: mockPriceService,
	}

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("global performance aggregates all accounts", prop.ForAll(
		func(numAssets int) bool {
			if numAssets <= 0 || numAssets > 10 {
				return true // Skip invalid or too large
			}

			var allTransactions []models.Transaction
			var expectedTotalValue float64
			var expectedTotalInvested float64
			var expectedTotalFees float64

			// Create transactions for multiple assets
			for i := 0; i < numAssets; i++ {
				isin := formatISIN(i)
				quantity := float64(10 + i)
				buyPrice := float64(100 + i*10)
				currentPrice := float64(110 + i*10)
				fees := float64(i)

				mockPriceService.SetPrice(isin, currentPrice)

				allTransactions = append(allTransactions, models.Transaction{
					ID:              formatTxID(i),
					AccountID:       formatAccountID(i % 3), // Distribute across 3 accounts
					ISIN:            stringPtr(isin),
					Quantity:        quantity,
					AmountValue:     quantity * buyPrice,
					TransactionType: "buy",
					Fees:            formatFees(fees),
					Timestamp:       time.Now().Format(time.RFC3339),
				})

				expectedTotalValue += quantity * currentPrice
				expectedTotalInvested += quantity * buyPrice
				expectedTotalFees += fees
			}

			// Calculate global performance
			startDate := time.Now().AddDate(0, 0, -7)
			endDate := time.Now()
			performance, err := service.calculatePerformance(allTransactions, startDate, endDate)
			if err != nil {
				t.Logf("calculatePerformance failed: %v", err)
				return false
			}

			// Verify aggregation
			if !floatEquals(performance.TotalValue, expectedTotalValue, 0.1) {
				t.Logf("Global TotalValue mismatch: got %v, want %v", performance.TotalValue, expectedTotalValue)
				return false
			}

			if !floatEquals(performance.TotalInvested, expectedTotalInvested, 0.1) {
				t.Logf("Global TotalInvested mismatch: got %v, want %v", performance.TotalInvested, expectedTotalInvested)
				return false
			}

			if !floatEquals(performance.TotalFees, expectedTotalFees, 0.1) {
				t.Logf("Global TotalFees mismatch: got %v, want %v", performance.TotalFees, expectedTotalFees)
				return false
			}

			return true
		},
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t)
}

// **Propriété 16: Calcul de valeur actuelle**
// **Valide: Exigences 10.7**
//
// Property: For all assets held in the portfolio, the current value must be calculated
// by multiplying the quantity held by the current price of the asset, and the sum of
// all current values must equal the total portfolio value.
func TestProperty_CurrentValueCalculation(t *testing.T) {
	mockPriceService := NewMockPriceService()
	service := &PerformanceService{
		DB:           nil,
		PriceService: mockPriceService,
	}

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("current value = quantity × current price", prop.ForAll(
		func(quantity float64, currentPrice float64) bool {
			if quantity <= 0 || currentPrice <= 0 {
				return true
			}

			isin := "TEST123456"
			mockPriceService.SetPrice(isin, currentPrice)

			transactions := []models.Transaction{
				{
					ID:              "tx1",
					AccountID:       "acc1",
					ISIN:            stringPtr(isin),
					Quantity:        quantity,
					AmountValue:     quantity * 100, // Buy price doesn't matter for this test
					TransactionType: "buy",
					Fees:            "0",
					Timestamp:       time.Now().Format(time.RFC3339),
				},
			}

			startDate := time.Now().AddDate(0, 0, -1)
			endDate := time.Now()
			performance, err := service.calculatePerformance(transactions, startDate, endDate)
			if err != nil {
				t.Logf("calculatePerformance failed: %v", err)
				return false
			}

			expectedValue := quantity * currentPrice
			if !floatEquals(performance.TotalValue, expectedValue, 0.01) {
				t.Logf("Current value mismatch: got %v, want %v (quantity=%v, price=%v)",
					performance.TotalValue, expectedValue, quantity, currentPrice)
				return false
			}

			return true
		},
		gen.Float64Range(1, 100),
		gen.Float64Range(10, 1000),
	))

	properties.Property("sum of asset values equals total portfolio value", prop.ForAll(
		func(numAssets int) bool {
			if numAssets <= 0 || numAssets > 10 {
				return true
			}

			var transactions []models.Transaction
			var expectedTotalValue float64

			for i := 0; i < numAssets; i++ {
				isin := formatISIN(i)
				quantity := float64(10 + i)
				currentPrice := float64(100 + i*10)

				mockPriceService.SetPrice(isin, currentPrice)

				transactions = append(transactions, models.Transaction{
					ID:              formatTxID(i),
					AccountID:       "acc1",
					ISIN:            stringPtr(isin),
					Quantity:        quantity,
					AmountValue:     quantity * 50, // Buy price
					TransactionType: "buy",
					Fees:            "0",
					Timestamp:       time.Now().Format(time.RFC3339),
				})

				expectedTotalValue += quantity * currentPrice
			}

			startDate := time.Now().AddDate(0, 0, -1)
			endDate := time.Now()
			performance, err := service.calculatePerformance(transactions, startDate, endDate)
			if err != nil {
				t.Logf("calculatePerformance failed: %v", err)
				return false
			}

			if !floatEquals(performance.TotalValue, expectedTotalValue, 0.1) {
				t.Logf("Total portfolio value mismatch: got %v, want %v", performance.TotalValue, expectedTotalValue)
				return false
			}

			return true
		},
		gen.IntRange(1, 10),
	))

	properties.Property("buy and sell transactions affect current value correctly", prop.ForAll(
		func(buyQuantity float64, sellQuantity float64, currentPrice float64) bool {
			if buyQuantity <= 0 || sellQuantity < 0 || sellQuantity > buyQuantity || currentPrice <= 0 {
				return true
			}

			isin := "TEST123456"
			mockPriceService.SetPrice(isin, currentPrice)

			transactions := []models.Transaction{
				{
					ID:              "tx1",
					AccountID:       "acc1",
					ISIN:            stringPtr(isin),
					Quantity:        buyQuantity,
					AmountValue:     buyQuantity * 100,
					TransactionType: "buy",
					Fees:            "0",
					Timestamp:       time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				},
			}

			if sellQuantity > 0 {
				transactions = append(transactions, models.Transaction{
					ID:              "tx2",
					AccountID:       "acc1",
					ISIN:            stringPtr(isin),
					Quantity:        sellQuantity,
					AmountValue:     sellQuantity * 100,
					TransactionType: "sell",
					Fees:            "0",
					Timestamp:       time.Now().Format(time.RFC3339),
				})
			}

			startDate := time.Now().AddDate(0, 0, -2)
			endDate := time.Now()
			performance, err := service.calculatePerformance(transactions, startDate, endDate)
			if err != nil {
				t.Logf("calculatePerformance failed: %v", err)
				return false
			}

			remainingQuantity := buyQuantity - sellQuantity
			expectedValue := remainingQuantity * currentPrice

			if !floatEquals(performance.TotalValue, expectedValue, 0.01) {
				t.Logf("Value after sell mismatch: got %v, want %v (remaining=%v, price=%v)",
					performance.TotalValue, expectedValue, remainingQuantity, currentPrice)
				return false
			}

			return true
		},
		gen.Float64Range(10, 100),
		gen.Float64Range(0, 50),
		gen.Float64Range(10, 1000),
	))

	properties.TestingRun(t)
}

// Helper functions

func floatEquals(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}

func formatFees(fees float64) string {
	return formatFloat(fees)
}

func formatFloat(f float64) string {
	return formatFloatWithPrecision(f, 2)
}

func formatFloatWithPrecision(f float64, precision int) string {
	format := "%." + formatInt(precision) + "f"
	return formatString(format, f)
}

func formatInt(i int) string {
	return string(rune('0' + i))
}

func formatString(format string, args ...interface{}) string {
	// Simple implementation for testing
	if len(args) == 0 {
		return format
	}
	// For float formatting
	if f, ok := args[0].(float64); ok {
		if format == "%.2f" {
			return formatFloatSimple(f)
		}
	}
	return format
}

func formatFloatSimple(f float64) string {
	// Simple float to string conversion
	intPart := int(f)
	fracPart := int((f - float64(intPart)) * 100)
	if fracPart < 0 {
		fracPart = -fracPart
	}

	result := ""
	if intPart < 0 {
		result = "-"
		intPart = -intPart
	}

	result += intToString(intPart) + "."
	if fracPart < 10 {
		result += "0"
	}
	result += intToString(fracPart)

	return result
}

func intToString(i int) string {
	if i == 0 {
		return "0"
	}

	digits := ""
	for i > 0 {
		digit := i % 10
		digits = string(rune('0'+digit)) + digits
		i /= 10
	}
	return digits
}

func formatISIN(i int) string {
	return "TEST" + padInt(i, 6)
}

func formatTxID(i int) string {
	return "tx" + padInt(i, 3)
}

func formatAccountID(i int) string {
	return "acc" + padInt(i, 2)
}

func padInt(i, width int) string {
	s := intToString(i)
	for len(s) < width {
		s = "0" + s
	}
	return s
}
