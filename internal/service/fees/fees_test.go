package fees

import (
	"fmt"
	"testing"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// setupTestDB creates a test database for testing
func setupTestDB(t *testing.T) *database.DB {
	cfg := database.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "valhafin",
		Password: "valhafin",
		DBName:   "valhafin_test",
		SSLMode:  "disable",
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return nil
	}

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// cleanupTestDB cleans up test data
func cleanupTestDB(t *testing.T, db *database.DB) {
	if db == nil {
		return
	}

	// Clean up test accounts and transactions
	_, err := db.Exec("DELETE FROM accounts WHERE name LIKE 'Test%'")
	if err != nil {
		t.Logf("Warning: failed to clean up test accounts: %v", err)
	}

	// Clean up transactions from all platform tables
	platforms := []string{"traderepublic", "binance", "boursedirect"}
	for _, platform := range platforms {
		tableName := fmt.Sprintf("transactions_%s", platform)
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE account_id LIKE 'test-%%'", tableName))
		if err != nil {
			t.Logf("Warning: failed to clean up %s: %v", tableName, err)
		}
	}
}

// **Propriété 17: Agrégation des frais**
// **Valide: Exigences 5.1, 5.2, 5.3, 5.4**
//
// Property: For all accounts or sets of accounts, the system must correctly calculate
// total fees, average fees per transaction, and breakdown by transaction type, and these
// metrics must be consistent with the stored transactions.
func TestProperty_FeesAggregation(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available")
		return
	}
	defer cleanupTestDB(t, db)

	service := NewFeesService(db)

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 30
	properties := gopter.NewProperties(parameters)

	properties.Property("fees aggregation is consistent with transactions", prop.ForAll(
		func(numTransactions int, feeValues []float64) bool {
			// Ensure valid inputs
			if numTransactions <= 0 || numTransactions > 20 {
				return true // Skip invalid inputs
			}

			if len(feeValues) == 0 {
				return true
			}

			// Limit fee values to reasonable range
			for i := range feeValues {
				if feeValues[i] < 0 {
					feeValues[i] = -feeValues[i]
				}
				if feeValues[i] > 100 {
					feeValues[i] = feeValues[i] / 100
				}
			}

			// Create test account
			account := &models.Account{
				Name:        "Test Fees Account",
				Platform:    "traderepublic",
				Credentials: "encrypted_test_credentials",
			}

			if err := db.CreateAccount(account); err != nil {
				t.Logf("Failed to create account: %v", err)
				return false
			}

			accountID := account.ID

			// Create test asset (required for foreign key constraint)
			_, errAsset := db.Exec(`
				INSERT INTO assets (isin, name, symbol, type, currency, last_updated)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (isin) DO NOTHING
			`, "TEST123456", "Test Asset", "TEST", "stock", "EUR", time.Now())
			if errAsset != nil {
				t.Logf("Failed to create asset: %v", errAsset)
				return false
			}

			// Create transactions with fees
			transactions := []models.Transaction{}
			expectedTotalFees := 0.0
			feesByType := make(map[string]float64)
			transactionTypes := []string{"buy", "sell", "dividend"}

			for i := 0; i < numTransactions; i++ {
				feeValue := feeValues[i%len(feeValues)]
				txType := transactionTypes[i%len(transactionTypes)]

				tx := models.Transaction{
					ID:              fmt.Sprintf("tx-%d-%d", time.Now().UnixNano(), i),
					AccountID:       accountID,
					Timestamp:       time.Now().AddDate(0, 0, -i).Format(time.RFC3339),
					Title:           fmt.Sprintf("Transaction %d", i),
					AmountValue:     100.0,
					AmountCurrency:  "EUR",
					Fees:            fmt.Sprintf("%.2f €", feeValue),
					TransactionType: txType,
					ISIN:            "TEST123456",
					Quantity:        1.0,
					Metadata:        "{}",
				}

				transactions = append(transactions, tx)
				expectedTotalFees += feeValue
				feesByType[txType] += feeValue
			}

			// Insert transactions
			if err := db.CreateTransactionsBatch(transactions, "traderepublic"); err != nil {
				t.Logf("Failed to create transactions: %v", err)
				return false
			}

			// Calculate fees using the service
			metrics, err := service.CalculateAccountFees(accountID, "", "")
			if err != nil {
				t.Logf("Failed to calculate fees: %v", err)
				return false
			}

			// Verify total fees (with small tolerance for floating point)
			tolerance := 0.01
			if abs(metrics.TotalFees-expectedTotalFees) > tolerance {
				t.Logf("Total fees mismatch: got %.2f, expected %.2f", metrics.TotalFees, expectedTotalFees)
				return false
			}

			// Verify transaction count
			if metrics.TransactionCount != numTransactions {
				t.Logf("Transaction count mismatch: got %d, expected %d", metrics.TransactionCount, numTransactions)
				return false
			}

			// Verify average fees
			expectedAverage := expectedTotalFees / float64(numTransactions)
			if abs(metrics.AverageFees-expectedAverage) > tolerance {
				t.Logf("Average fees mismatch: got %.2f, expected %.2f", metrics.AverageFees, expectedAverage)
				return false
			}

			// Verify fees by type
			for txType, expectedFees := range feesByType {
				actualFees, exists := metrics.FeesByType[txType]
				if !exists {
					t.Logf("Missing fees for type %s", txType)
					return false
				}
				if abs(actualFees-expectedFees) > tolerance {
					t.Logf("Fees by type mismatch for %s: got %.2f, expected %.2f", txType, actualFees, expectedFees)
					return false
				}
			}

			// Verify time series is not empty
			if len(metrics.TimeSeries) == 0 {
				t.Logf("Time series is empty")
				return false
			}

			// Verify time series total matches total fees
			timeSeriesTotal := 0.0
			for _, point := range metrics.TimeSeries {
				timeSeriesTotal += point.Fees
			}
			if abs(timeSeriesTotal-expectedTotalFees) > tolerance {
				t.Logf("Time series total mismatch: got %.2f, expected %.2f", timeSeriesTotal, expectedTotalFees)
				return false
			}

			// Clean up
			db.DeleteAccount(accountID)

			return true
		},
		gen.IntRange(1, 20),
		gen.SliceOfN(20, gen.Float64Range(0.1, 10.0)),
	))

	properties.TestingRun(t)
}

// TestProperty_GlobalFeesAggregation tests global fees aggregation across multiple accounts
func TestProperty_GlobalFeesAggregation(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available")
		return
	}
	defer cleanupTestDB(t, db)

	service := NewFeesService(db)

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20
	properties := gopter.NewProperties(parameters)

	properties.Property("global fees aggregation sums all accounts", prop.ForAll(
		func(numAccounts int, feesPerAccount []float64) bool {
			// Ensure valid inputs
			if numAccounts <= 0 || numAccounts > 5 {
				return true // Skip invalid inputs
			}

			if len(feesPerAccount) == 0 {
				return true
			}

			// Limit fee values to reasonable range
			for i := range feesPerAccount {
				if feesPerAccount[i] < 0 {
					feesPerAccount[i] = -feesPerAccount[i]
				}
				if feesPerAccount[i] > 50 {
					feesPerAccount[i] = feesPerAccount[i] / 10
				}
			}

			accountIDs := []string{}
			expectedTotalFees := 0.0
			expectedTotalTransactions := 0

			// Create multiple accounts with transactions
			for i := 0; i < numAccounts; i++ {
				account := &models.Account{
					Name:        fmt.Sprintf("Test Global Account %d", i),
					Platform:    "traderepublic",
					Credentials: "encrypted_test_credentials",
				}

				if err := db.CreateAccount(account); err != nil {
					t.Logf("Failed to create account: %v", err)
					return false
				}

				accountID := account.ID
				accountIDs = append(accountIDs, accountID)

				// Create test asset (required for foreign key constraint)
				_, errAsset := db.Exec(`
					INSERT INTO assets (isin, name, symbol, type, currency, last_updated)
					VALUES ($1, $2, $3, $4, $5, $6)
					ON CONFLICT (isin) DO NOTHING
				`, "TEST123456", "Test Asset", "TEST", "stock", "EUR", time.Now())
				if errAsset != nil {
					t.Logf("Failed to create asset: %v", errAsset)
					return false
				}

				// Create transactions for this account
				feeValue := feesPerAccount[i%len(feesPerAccount)]
				numTx := 3 // Fixed number of transactions per account

				for j := 0; j < numTx; j++ {
					tx := models.Transaction{
						ID:              fmt.Sprintf("tx-global-%d-%d-%d", time.Now().UnixNano(), i, j),
						AccountID:       accountID,
						Timestamp:       time.Now().AddDate(0, 0, -j).Format(time.RFC3339),
						Title:           fmt.Sprintf("Transaction %d", j),
						AmountValue:     100.0,
						AmountCurrency:  "EUR",
						Fees:            fmt.Sprintf("%.2f €", feeValue),
						TransactionType: "buy",
						ISIN:            "TEST123456",
						Quantity:        1.0,
						Metadata:        "{}",
					}

					if err := db.CreateTransaction(&tx, "traderepublic"); err != nil {
						t.Logf("Failed to create transaction: %v", err)
						return false
					}

					expectedTotalFees += feeValue
					expectedTotalTransactions++
				}
			}

			// Calculate global fees
			metrics, err := service.CalculateGlobalFees("", "")
			if err != nil {
				t.Logf("Failed to calculate global fees: %v", err)
				return false
			}

			// Verify total fees
			tolerance := 0.01
			if abs(metrics.TotalFees-expectedTotalFees) > tolerance {
				t.Logf("Global total fees mismatch: got %.2f, expected %.2f", metrics.TotalFees, expectedTotalFees)
				return false
			}

			// Verify transaction count
			if metrics.TransactionCount != expectedTotalTransactions {
				t.Logf("Global transaction count mismatch: got %d, expected %d", metrics.TransactionCount, expectedTotalTransactions)
				return false
			}

			// Clean up
			for _, accountID := range accountIDs {
				db.DeleteAccount(accountID)
			}

			return true
		},
		gen.IntRange(1, 5),
		gen.SliceOfN(10, gen.Float64Range(0.5, 5.0)),
	))

	properties.TestingRun(t)
}

// TestProperty_FeesFilteringByPeriod tests that fees are correctly filtered by date range
func TestProperty_FeesFilteringByPeriod(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available")
		return
	}
	defer cleanupTestDB(t, db)

	service := NewFeesService(db)

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20
	properties := gopter.NewProperties(parameters)

	properties.Property("fees are correctly filtered by date range", prop.ForAll(
		func(daysBack int) bool {
			// Ensure valid inputs
			if daysBack <= 0 || daysBack > 365 {
				return true // Skip invalid inputs
			}

			// Create test account
			account := &models.Account{
				Name:        "Test Period Account",
				Platform:    "traderepublic",
				Credentials: "encrypted_test_credentials",
			}

			if err := db.CreateAccount(account); err != nil {
				t.Logf("Failed to create account: %v", err)
				return false
			}

			accountID := account.ID

			// Create test asset (required for foreign key constraint)
			_, errAsset := db.Exec(`
				INSERT INTO assets (isin, name, symbol, type, currency, last_updated)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (isin) DO NOTHING
			`, "TEST123456", "Test Asset", "TEST", "stock", "EUR", time.Now())
			if errAsset != nil {
				t.Logf("Failed to create asset: %v", errAsset)
				return false
			}

			// Create transactions spanning different dates
			now := time.Now()
			feeValue := 1.0

			// Create transactions at specific day offsets
			dayOffsets := []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90}

			for _, offset := range dayOffsets {
				txDate := now.AddDate(0, 0, -offset)
				tx := models.Transaction{
					ID:              fmt.Sprintf("tx-period-%d-%d", time.Now().UnixNano(), offset),
					AccountID:       accountID,
					Timestamp:       txDate.Format(time.RFC3339),
					Title:           fmt.Sprintf("Transaction %d", offset),
					AmountValue:     100.0,
					AmountCurrency:  "EUR",
					Fees:            fmt.Sprintf("%.2f €", feeValue),
					TransactionType: "buy",
					ISIN:            "TEST123456",
					Quantity:        1.0,
					Metadata:        "{}",
				}

				if err := db.CreateTransaction(&tx, "traderepublic"); err != nil {
					t.Logf("Failed to create transaction: %v", err)
					return false
				}
			}

			// Calculate expected fees in range
			// Count how many offsets are <= daysBack
			expectedTransactionsInRange := 0
			for _, offset := range dayOffsets {
				if offset <= daysBack {
					expectedTransactionsInRange++
				}
			}
			expectedFeesInRange := float64(expectedTransactionsInRange) * feeValue

			// Calculate fees with date filter
			startDate := now.AddDate(0, 0, -daysBack).Format("2006-01-02")
			endDate := now.Format("2006-01-02")

			metrics, err := service.CalculateAccountFees(accountID, startDate, endDate)
			if err != nil {
				t.Logf("Failed to calculate fees: %v", err)
				return false
			}

			// Verify that only transactions in range are counted
			tolerance := 0.01
			if abs(metrics.TotalFees-expectedFeesInRange) > tolerance {
				t.Logf("Filtered fees mismatch: got %.2f, expected %.2f (daysBack=%d, transactions in range=%d)",
					metrics.TotalFees, expectedFeesInRange, daysBack, expectedTransactionsInRange)
				return false
			}

			if metrics.TransactionCount != expectedTransactionsInRange {
				t.Logf("Filtered transaction count mismatch: got %d, expected %d", metrics.TransactionCount, expectedTransactionsInRange)
				return false
			}

			// Clean up
			db.DeleteAccount(accountID)

			return true
		},
		gen.IntRange(30, 180),
	))

	properties.TestingRun(t)
}

// Helper function to calculate absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Unit tests for helper functions

func TestParseFeeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"empty string", "", 0},
		{"simple float", "1.23", 1.23},
		{"with euro symbol", "1,50 €", 1.50},
		{"with dollar symbol", "2.75 $", 2.75},
		{"with USD", "3.00 USD", 3.00},
		{"with EUR", "4,25 EUR", 4.25},
		{"negative value", "-1.50", 1.50}, // Should return absolute value
		{"zero", "0", 0},
		{"large value", "123.45", 123.45},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFeeValue(tt.input)
			if abs(result-tt.expected) > 0.001 {
				t.Errorf("parseFeeValue(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractDate(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		expected  string
	}{
		{"RFC3339 format", "2024-01-15T10:30:00Z", "2024-01-15"},
		{"already date format", "2024-01-15", "2024-01-15"},
		{"empty string", "", ""},
		{"invalid format", "invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDate(tt.timestamp)
			if result != tt.expected {
				t.Errorf("extractDate(%q) = %v, want %v", tt.timestamp, result, tt.expected)
			}
		})
	}
}

func TestSortTimeSeries(t *testing.T) {
	series := []FeeTimeSeriesPoint{
		{Date: "2024-01-15", Fees: 1.0},
		{Date: "2024-01-10", Fees: 2.0},
		{Date: "2024-01-20", Fees: 3.0},
		{Date: "2024-01-05", Fees: 4.0},
	}

	sortTimeSeries(series)

	// Verify sorted order
	expected := []string{"2024-01-05", "2024-01-10", "2024-01-15", "2024-01-20"}
	for i, point := range series {
		if point.Date != expected[i] {
			t.Errorf("sortTimeSeries: position %d = %v, want %v", i, point.Date, expected[i])
		}
	}
}
