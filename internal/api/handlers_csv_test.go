package api

import (
	"bytes"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"
	encryptionsvc "valhafin/internal/service/encryption"
	"valhafin/internal/service/fees"
	"valhafin/internal/service/performance"
	"valhafin/internal/service/price"
	"valhafin/internal/service/sync"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// setupTestHandlerForCSV creates a test handler with dependencies for CSV tests
func setupTestHandlerForCSV(t *testing.T) (*Handler, *database.DB) {
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
		return nil, nil
	}

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create encryption service with test key
	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate encryption key: %v", err)
	}

	encryptionService, err := encryptionsvc.NewEncryptionService(key)
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	// Create scraper factory and sync service
	scraperFactory := sync.NewScraperFactory()
	syncService := sync.NewService(db, scraperFactory, encryptionService)

	// Create price service
	priceService := price.NewYahooFinanceService(db)

	// Create performance service
	performanceService := performance.NewPerformanceService(db, priceService)

	// Create fees service
	feesService := fees.NewFeesService(db)

	handler := NewHandler(db, encryptionService, syncService, priceService, performanceService, feesService)
	return handler, db
}

// cleanupTestDBForCSV cleans up test data
func cleanupTestDBForCSV(t *testing.T, db *database.DB) {
	if db == nil {
		return
	}

	// Clean up all test data
	_, _ = db.Exec("DELETE FROM transactions_traderepublic")
	_, _ = db.Exec("DELETE FROM transactions_binance")
	_, _ = db.Exec("DELETE FROM transactions_boursedirect")
	_, _ = db.Exec("DELETE FROM asset_prices")
	_, _ = db.Exec("DELETE FROM assets")
	_, _ = db.Exec("DELETE FROM accounts")
}

// createTestAccount creates a test account and returns its ID
func createTestAccount(t *testing.T, db *database.DB, platform string) string {
	account := &models.Account{
		Name:        fmt.Sprintf("Test %s Account", platform),
		Platform:    platform,
		Credentials: "encrypted_test_credentials",
	}

	if err := db.CreateAccount(account); err != nil {
		t.Fatalf("Failed to create test account: %v", err)
	}

	return account.ID
}

// createCSVMultipartRequest creates a multipart form request with a CSV file
func createCSVMultipartRequest(accountID string, csvContent string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add account_id field
	if err := writer.WriteField("account_id", accountID); err != nil {
		return nil, err
	}

	// Add CSV file
	part, err := writer.CreateFormFile("file", "transactions.csv")
	if err != nil {
		return nil, err
	}

	if _, err := io.WriteString(part, csvContent); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest("POST", "/api/transactions/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

// generateCSVContent generates CSV content from transactions
func generateCSVContent(transactions []models.Transaction, includeHeader bool) string {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if includeHeader {
		header := []string{
			"id", "timestamp", "isin", "amount_value", "fees",
			"title", "amount_currency", "quantity", "transaction_type",
		}
		writer.Write(header)
	}

	for _, t := range transactions {
		isin := ""
		if t.ISIN != nil {
			isin = *t.ISIN
		}
		row := []string{
			t.ID,
			t.Timestamp,
			isin,
			fmt.Sprintf("%.2f", t.AmountValue),
			t.Fees,
			t.Title,
			t.AmountCurrency,
			fmt.Sprintf("%.2f", t.Quantity),
			t.TransactionType,
		}
		writer.Write(row)
	}

	writer.Flush()
	return buf.String()
}

// **Propriété 20: Parsing et validation CSV**
// **Valide: Exigences 9.1, 9.2, 9.3**
//
// For all CSV files imported, the system must validate the presence of required columns
// (timestamp, isin, amount_value, fees) before parsing, and if the file is invalid,
// reject the import with a detailed error report listing all errors found.
func TestProperty20_CSVParsingAndValidation(t *testing.T) {
	handler, db := setupTestHandlerForCSV(t)
	if handler == nil {
		return
	}
	defer cleanupTestDBForCSV(t, db)
	defer db.Close()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 30
	properties := gopter.NewProperties(parameters)

	// Property: CSV without required columns is rejected
	properties.Property("CSV without required columns is rejected", prop.ForAll(
		func(missingColumn string) bool {
			// Create test account
			accountID := createTestAccount(t, db, "traderepublic")
			defer db.DeleteAccount(accountID)

			// Create CSV with missing column
			var csvContent string
			switch missingColumn {
			case "timestamp":
				csvContent = "id,isin,amount_value,fees\n1,US0378331005,100.00,1.00\n"
			case "isin":
				csvContent = "id,timestamp,amount_value,fees\n1,2024-01-01T00:00:00Z,100.00,1.00\n"
			case "amount_value":
				csvContent = "id,timestamp,isin,fees\n1,2024-01-01T00:00:00Z,US0378331005,1.00\n"
			case "fees":
				csvContent = "id,timestamp,isin,amount_value\n1,2024-01-01T00:00:00Z,US0378331005,100.00\n"
			}

			// Create request
			req, err := createCSVMultipartRequest(accountID, csvContent)
			if err != nil {
				t.Logf("Failed to create request: %v", err)
				return false
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler.ImportCSVHandler(rr, req)

			// Should return 400 Bad Request
			if rr.Code != http.StatusBadRequest {
				t.Logf("Expected status 400, got %d", rr.Code)
				return false
			}

			// Response should contain error about missing column
			body := rr.Body.String()
			if !strings.Contains(body, missingColumn) {
				t.Logf("Expected error message to mention missing column '%s', got: %s", missingColumn, body)
				return false
			}

			return true
		},
		gen.OneConstOf("timestamp", "isin", "amount_value", "fees"),
	))

	// Property: Valid CSV with all required columns is accepted
	properties.Property("valid CSV with required columns is accepted", prop.ForAll(
		func(numRows int) bool {
			if numRows < 1 {
				numRows = 1
			}
			if numRows > 10 {
				numRows = 10
			}

			// Create test account
			accountID := createTestAccount(t, db, "traderepublic")
			defer db.DeleteAccount(accountID)

			// Create test asset
			asset := &models.Asset{
				ISIN:     "US0378331005",
				Name:     "Apple Inc.",
				Symbol:   stringPtr("AAPL"),
				Type:     "stock",
				Currency: "USD",
			}
			db.CreateAsset(asset)
			defer db.Exec("DELETE FROM assets WHERE isin = $1", asset.ISIN)

			// Create valid CSV content
			var buf bytes.Buffer
			writer := csv.NewWriter(&buf)

			// Write header
			header := []string{"id", "timestamp", "isin", "amount_value", "fees", "amount_currency", "title"}
			writer.Write(header)

			// Write rows
			for i := 0; i < numRows; i++ {
				row := []string{
					fmt.Sprintf("txn_%d", i),
					time.Now().Add(time.Duration(-i) * time.Hour).Format(time.RFC3339),
					"US0378331005",
					fmt.Sprintf("%.2f", 100.0+float64(i)),
					"1.50",
					"EUR",
					fmt.Sprintf("Transaction %d", i),
				}
				writer.Write(row)
			}
			writer.Flush()

			// Create request
			req, err := createCSVMultipartRequest(accountID, buf.String())
			if err != nil {
				t.Logf("Failed to create request: %v", err)
				return false
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler.ImportCSVHandler(rr, req)

			// Should return 200 OK
			if rr.Code != http.StatusOK {
				t.Logf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
				return false
			}

			// Parse response
			var summary ImportSummary
			if err := json.NewDecoder(rr.Body).Decode(&summary); err != nil {
				t.Logf("Failed to decode response: %v", err)
				return false
			}

			// Log the summary for debugging
			t.Logf("Import summary: imported=%d, ignored=%d, errors=%d, details=%v",
				summary.Imported, summary.Ignored, summary.Errors, summary.Details)

			// Should have imported transactions
			if summary.Imported != numRows {
				t.Logf("Expected %d imported, got %d", numRows, summary.Imported)
				return false
			}

			return true
		},
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t)
}

// **Propriété 21: Import CSV avec déduplication**
// **Valide: Exigences 9.4, 9.5, 9.6**
//
// For all valid CSV files imported, the system must insert transactions into the account's
// corresponding table, detect and ignore duplicate transactions (based on id or combination
// of date+asset+amount), and return a summary with the number of imported, ignored, and error transactions.
func TestProperty21_CSVImportWithDeduplication(t *testing.T) {
	handler, db := setupTestHandlerForCSV(t)
	if handler == nil {
		return
	}
	defer cleanupTestDBForCSV(t, db)
	defer db.Close()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 30
	properties := gopter.NewProperties(parameters)

	// Generator for transactions
	genTransaction := func(id string) models.Transaction {
		return models.Transaction{
			ID:              id,
			Timestamp:       time.Now().Format(time.RFC3339),
			ISIN:            stringPtr("US0378331005"),
			AmountValue:     100.50,
			Fees:            "1.50",
			AmountCurrency:  "EUR",
			Title:           "Test Transaction",
			Quantity:        10.0,
			TransactionType: "buy",
		}
	}

	properties.Property("duplicate transactions are ignored", prop.ForAll(
		func(numTransactions int) bool {
			if numTransactions < 2 {
				numTransactions = 2
			}
			if numTransactions > 20 {
				numTransactions = 20
			}

			// Create test account
			accountID := createTestAccount(t, db, "traderepublic")
			defer db.DeleteAccount(accountID)

			// Create test asset
			asset := &models.Asset{
				ISIN:     "US0378331005",
				Name:     "Apple Inc.",
				Symbol:   stringPtr("AAPL"),
				Type:     "stock",
				Currency: "USD",
			}
			db.CreateAsset(asset)
			defer db.Exec("DELETE FROM assets WHERE isin = $1", asset.ISIN)

			// Create transactions with some duplicates
			transactions := []models.Transaction{}
			for i := 0; i < numTransactions; i++ {
				t := genTransaction(fmt.Sprintf("txn_%d", i))
				t.AccountID = accountID
				transactions = append(transactions, t)
			}

			// Add duplicate of first transaction
			duplicate := transactions[0]
			transactions = append(transactions, duplicate)

			// Generate CSV content
			csvContent := generateCSVContent(transactions, true)

			// Create request
			req, err := createCSVMultipartRequest(accountID, csvContent)
			if err != nil {
				t.Logf("Failed to create request: %v", err)
				return false
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler.ImportCSVHandler(rr, req)

			// Should return 200 OK
			if rr.Code != http.StatusOK {
				t.Logf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
				return false
			}

			// Parse response
			var summary ImportSummary
			if err := json.NewDecoder(rr.Body).Decode(&summary); err != nil {
				t.Logf("Failed to decode response: %v", err)
				return false
			}

			// Should have imported numTransactions and ignored 1 duplicate
			expectedImported := numTransactions
			expectedIgnored := 1

			if summary.Imported != expectedImported {
				t.Logf("Expected %d imported, got %d", expectedImported, summary.Imported)
				return false
			}

			if summary.Ignored != expectedIgnored {
				t.Logf("Expected %d ignored, got %d", expectedIgnored, summary.Ignored)
				return false
			}

			return true
		},
		gen.IntRange(2, 20),
	))

	properties.Property("import summary is accurate", prop.ForAll(
		func(numValid int, numInvalid int) bool {
			if numValid < 1 {
				numValid = 1
			}
			if numValid > 10 {
				numValid = 10
			}
			if numInvalid < 0 {
				numInvalid = 0
			}
			if numInvalid > 5 {
				numInvalid = 5
			}

			// Create test account
			accountID := createTestAccount(t, db, "traderepublic")
			defer db.DeleteAccount(accountID)

			// Create test asset
			asset := &models.Asset{
				ISIN:     "US0378331005",
				Name:     "Apple Inc.",
				Symbol:   stringPtr("AAPL"),
				Type:     "stock",
				Currency: "USD",
			}
			db.CreateAsset(asset)
			defer db.Exec("DELETE FROM assets WHERE isin = $1", asset.ISIN)

			// Create valid transactions
			transactions := []models.Transaction{}
			for i := 0; i < numValid; i++ {
				t := genTransaction(fmt.Sprintf("valid_txn_%d", i))
				t.AccountID = accountID
				transactions = append(transactions, t)
			}

			// Create invalid transactions (missing required fields)
			for i := 0; i < numInvalid; i++ {
				t := models.Transaction{
					ID:        fmt.Sprintf("invalid_txn_%d", i),
					AccountID: accountID,
					// Missing timestamp, ISIN, etc.
				}
				transactions = append(transactions, t)
			}

			// Generate CSV content
			csvContent := generateCSVContent(transactions, true)

			// Create request
			req, err := createCSVMultipartRequest(accountID, csvContent)
			if err != nil {
				t.Logf("Failed to create request: %v", err)
				return false
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler.ImportCSVHandler(rr, req)

			// Should return 200 OK (even with some errors)
			if rr.Code != http.StatusOK {
				t.Logf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
				return false
			}

			// Parse response
			var summary ImportSummary
			if err := json.NewDecoder(rr.Body).Decode(&summary); err != nil {
				t.Logf("Failed to decode response: %v", err)
				return false
			}

			// Should have imported valid transactions
			if summary.Imported != numValid {
				t.Logf("Expected %d imported, got %d", numValid, summary.Imported)
				return false
			}

			// Should have errors for invalid transactions
			if numInvalid > 0 && summary.Errors == 0 {
				t.Logf("Expected some errors for invalid transactions, got 0")
				return false
			}

			return true
		},
		gen.IntRange(1, 10),
		gen.IntRange(0, 5),
	))

	properties.TestingRun(t)
}
