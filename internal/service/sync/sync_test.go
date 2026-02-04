package sync

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"
	"valhafin/internal/service/encryption"
	"valhafin/internal/service/scraper/types"
)

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// setupTestDB creates a test database connection
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

	// Clean up test transactions first (due to foreign key constraints)
	platforms := []string{"traderepublic", "binance", "boursedirect"}
	for _, platform := range platforms {
		tableName := "transactions_" + platform
		query := fmt.Sprintf(`
			DELETE FROM %s 
			WHERE account_id IN (
				SELECT id FROM accounts WHERE name LIKE 'Test%%'
			)
		`, tableName)
		_, err := db.Exec(query)
		if err != nil {
			// Silently ignore errors during cleanup
			_ = err
		}
	}

	// Clean up test accounts
	_, err := db.Exec("DELETE FROM accounts WHERE name LIKE 'Test%'")
	if err != nil {
		// Silently ignore errors during cleanup
		_ = err
	}
}

// setupTestEncryption creates a test encryption service
func setupTestEncryption(t *testing.T) *encryption.EncryptionService {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate encryption key: %v", err)
	}

	encryptionService, err := encryption.NewEncryptionService(key)
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	return encryptionService
}

// mockScraper is a mock implementation of the Scraper interface for testing
type mockScraper struct {
	platform     string
	transactions []models.Transaction
	shouldError  bool
	errorType    string
}

func (m *mockScraper) GetPlatformName() string {
	return m.platform
}

func (m *mockScraper) ValidateCredentials(credentials map[string]interface{}) error {
	if m.shouldError && m.errorType == "validation" {
		return types.NewValidationError(m.platform, "Mock validation error", nil)
	}
	return nil
}

func (m *mockScraper) FetchTransactions(credentials map[string]interface{}, lastSync *time.Time) ([]models.Transaction, error) {
	if m.shouldError {
		switch m.errorType {
		case "auth":
			return nil, types.NewAuthError(m.platform, "Mock auth error", nil)
		case "network":
			return nil, types.NewNetworkError(m.platform, "Mock network error", nil)
		case "parsing":
			return nil, types.NewParsingError(m.platform, "Mock parsing error", nil)
		default:
			return nil, types.NewValidationError(m.platform, "Mock error", nil)
		}
	}

	// Filter transactions by lastSync if provided (incremental sync)
	if lastSync != nil {
		filtered := []models.Transaction{}
		for _, tx := range m.transactions {
			txTime, err := time.Parse(time.RFC3339, tx.Timestamp)
			if err == nil && txTime.After(*lastSync) {
				filtered = append(filtered, tx)
			}
		}
		return filtered, nil
	}

	return m.transactions, nil
}

// mockScraperFactory is a mock factory for testing
type mockScraperFactory struct {
	scrapers map[string]types.Scraper
}

func newMockScraperFactory() *mockScraperFactory {
	return &mockScraperFactory{
		scrapers: make(map[string]types.Scraper),
	}
}

func (f *mockScraperFactory) AddScraper(platform string, scraper types.Scraper) {
	f.scrapers[platform] = scraper
}

func (f *mockScraperFactory) GetScraper(platform string) (types.Scraper, error) {
	scraper, ok := f.scrapers[platform]
	if !ok {
		return nil, types.NewValidationError(platform, "Unsupported platform", nil)
	}
	return scraper, nil
}

// **Propriété 4: Synchronisation complète initiale**
// **Valide: Exigences 2.1, 2.2, 2.3**
func TestProperty4_FullInitialSync(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTestDB(t, db)
	defer db.Close()

	encryptionService := setupTestEncryption(t)

	// Test with different numbers of transactions
	testCases := []int{1, 5, 10}

	for _, numTransactions := range testCases {
		t.Run("Full sync with transactions", func(t *testing.T) {
			// Create a test asset first to satisfy foreign key constraint
			testISIN := "TEST00000001" // 12 characters max
			_, err := db.Exec(`
				INSERT INTO assets (isin, name, symbol, type, currency)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (isin) DO NOTHING
			`, testISIN, "Test Asset", "TEST", "stock", "EUR")
			if err != nil {
				t.Fatalf("Failed to create test asset: %v", err)
			}

			// Generate test transactions with valid ISIN
			transactions := make([]models.Transaction, numTransactions)
			for i := 0; i < numTransactions; i++ {
				transactions[i] = models.Transaction{
					ID:              "tx-full-" + time.Now().Format("20060102150405") + "-" + string(rune('0'+i)),
					Timestamp:       time.Now().Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
					Title:           "Test Transaction",
					AmountCurrency:  "EUR",
					AmountValue:     100.0,
					TransactionType: "buy",
					ISIN:            stringPtr(testISIN),
					Metadata:        stringPtr("{}"),
				}
			}

			// Create mock scraper
			mockFactory := newMockScraperFactory()
			mockFactory.AddScraper("traderepublic", &mockScraper{
				platform:     "traderepublic",
				transactions: transactions,
				shouldError:  false,
			})

			// Create sync service
			syncService := NewService(db, mockFactory, encryptionService)

			// Create test account
			credentials := map[string]interface{}{
				"phone_number": "+33612345678",
				"pin":          "1234",
			}
			credentialsJSON, _ := json.Marshal(credentials)
			encryptedCreds, _ := encryptionService.Encrypt(string(credentialsJSON))

			account := &models.Account{
				Name:        "Test Account Full",
				Platform:    "traderepublic",
				Credentials: encryptedCreds,
			}

			if err := db.CreateAccount(account); err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}
			defer db.DeleteAccount(account.ID)

			// Update transaction account IDs
			for i := range transactions {
				transactions[i].AccountID = account.ID
			}

			// Perform sync
			result, err := syncService.SyncAccount(account.ID)
			if err != nil {
				t.Fatalf("Sync failed: %v", err)
			}

			// Verify sync result
			if result.SyncType != "full" {
				t.Errorf("Expected full sync, got %s", result.SyncType)
			}

			if result.TransactionsFetched != numTransactions {
				t.Errorf("Expected %d transactions fetched, got %d", numTransactions, result.TransactionsFetched)
			}

			if result.TransactionsStored != numTransactions {
				t.Errorf("Expected %d transactions stored, got %d", numTransactions, result.TransactionsStored)
			}

			// Verify transactions are stored in database
			storedTransactions, err := db.GetTransactionsByAccount(account.ID, "traderepublic", database.TransactionFilter{})
			if err != nil {
				t.Fatalf("Failed to retrieve transactions: %v", err)
			}

			if len(storedTransactions) != numTransactions {
				t.Errorf("Expected %d stored transactions, got %d", numTransactions, len(storedTransactions))
			}
		})
	}
}

// **Propriété 5: Synchronisation incrémentale**
// **Valide: Exigences 2.4**
func TestProperty5_IncrementalSync(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTestDB(t, db)
	defer db.Close()

	encryptionService := setupTestEncryption(t)

	now := time.Now()
	lastSync := now.Add(-24 * time.Hour)

	// Create a test asset first to satisfy foreign key constraint
	testISIN := "TEST00000002" // 12 characters max
	_, err := db.Exec(`
		INSERT INTO assets (isin, name, symbol, type, currency)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (isin) DO NOTHING
	`, testISIN, "Test Asset 2", "TEST2", "stock", "EUR")
	if err != nil {
		t.Fatalf("Failed to create test asset: %v", err)
	}

	// Generate initial transactions (before lastSync) with valid ISIN
	initialTransactions := []models.Transaction{
		{
			ID:              "tx-old-1",
			Timestamp:       lastSync.Add(-2 * time.Hour).Format(time.RFC3339),
			Title:           "Old Transaction 1",
			AmountCurrency:  "EUR",
			AmountValue:     100.0,
			TransactionType: "buy",
			ISIN:            stringPtr(testISIN),
			Metadata:        stringPtr("{}"),
		},
		{
			ID:              "tx-old-2",
			Timestamp:       lastSync.Add(-1 * time.Hour).Format(time.RFC3339),
			Title:           "Old Transaction 2",
			AmountCurrency:  "EUR",
			AmountValue:     200.0,
			TransactionType: "buy",
			ISIN:            stringPtr(testISIN),
			Metadata:        stringPtr("{}"),
		},
	}

	// Generate new transactions (after lastSync) with valid ISIN
	newTransactions := []models.Transaction{
		{
			ID:              "tx-new-1",
			Timestamp:       now.Add(-1 * time.Hour).Format(time.RFC3339),
			Title:           "New Transaction 1",
			AmountCurrency:  "EUR",
			AmountValue:     150.0,
			TransactionType: "buy",
			ISIN:            stringPtr(testISIN),
			Metadata:        stringPtr("{}"),
		},
		{
			ID:              "tx-new-2",
			Timestamp:       now.Add(-30 * time.Minute).Format(time.RFC3339),
			Title:           "New Transaction 2",
			AmountCurrency:  "EUR",
			AmountValue:     250.0,
			TransactionType: "sell",
			ISIN:            stringPtr(testISIN),
			Metadata:        stringPtr("{}"),
		},
	}

	allTransactions := append(initialTransactions, newTransactions...)

	// Create mock scraper
	mockFactory := newMockScraperFactory()
	mockFactory.AddScraper("traderepublic", &mockScraper{
		platform:     "traderepublic",
		transactions: allTransactions,
		shouldError:  false,
	})

	// Create sync service
	syncService := NewService(db, mockFactory, encryptionService)

	// Create test account with lastSync set
	credentials := map[string]interface{}{
		"phone_number": "+33612345678",
		"pin":          "1234",
	}
	credentialsJSON, _ := json.Marshal(credentials)
	encryptedCreds, _ := encryptionService.Encrypt(string(credentialsJSON))

	account := &models.Account{
		Name:        "Test Account Incremental",
		Platform:    "traderepublic",
		Credentials: encryptedCreds,
		LastSync:    &lastSync,
	}

	if err := db.CreateAccount(account); err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}
	defer db.DeleteAccount(account.ID)

	// Update transaction account IDs
	for i := range allTransactions {
		allTransactions[i].AccountID = account.ID
	}
	for i := range initialTransactions {
		initialTransactions[i].AccountID = account.ID
	}
	for i := range newTransactions {
		newTransactions[i].AccountID = account.ID
	}

	// Store initial transactions
	if err := db.CreateTransactionsBatch(initialTransactions, "traderepublic"); err != nil {
		t.Fatalf("Failed to store initial transactions: %v", err)
	}

	// Perform incremental sync
	result, err := syncService.SyncAccount(account.ID)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify sync result
	if result.SyncType != "incremental" {
		t.Errorf("Expected incremental sync, got %s", result.SyncType)
	}

	if result.TransactionsFetched != len(newTransactions) {
		t.Errorf("Expected %d new transactions fetched, got %d", len(newTransactions), result.TransactionsFetched)
	}

	// Verify no duplicates
	storedTransactions, err := db.GetTransactionsByAccount(account.ID, "traderepublic", database.TransactionFilter{})
	if err != nil {
		t.Fatalf("Failed to retrieve transactions: %v", err)
	}

	expectedTotal := len(initialTransactions) + len(newTransactions)
	if len(storedTransactions) != expectedTotal {
		t.Errorf("Expected %d total transactions, got %d", expectedTotal, len(storedTransactions))
	}
}

// **Propriété 6: Gestion d'erreur de synchronisation**
// **Valide: Exigences 2.5**
func TestProperty6_SyncErrorHandling(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTestDB(t, db)
	defer db.Close()

	encryptionService := setupTestEncryption(t)

	errorTypes := []string{"auth", "network", "parsing", "validation"}

	for _, errorType := range errorTypes {
		t.Run("Error type: "+errorType, func(t *testing.T) {
			// Create mock scraper that returns an error
			mockFactory := newMockScraperFactory()
			mockFactory.AddScraper("traderepublic", &mockScraper{
				platform:    "traderepublic",
				shouldError: true,
				errorType:   errorType,
			})

			// Create sync service
			syncService := NewService(db, mockFactory, encryptionService)

			// Create test account
			credentials := map[string]interface{}{
				"phone_number": "+33612345678",
				"pin":          "1234",
			}
			credentialsJSON, _ := json.Marshal(credentials)
			encryptedCreds, _ := encryptionService.Encrypt(string(credentialsJSON))

			account := &models.Account{
				Name:        "Test Account Error",
				Platform:    "traderepublic",
				Credentials: encryptedCreds,
			}

			if err := db.CreateAccount(account); err != nil {
				t.Fatalf("Failed to create account: %v", err)
			}
			defer db.DeleteAccount(account.ID)

			// Perform sync (should fail)
			result, err := syncService.SyncAccount(account.ID)

			// Verify error is returned
			if err == nil {
				t.Error("Expected error, got nil")
			}

			// Verify result contains error information
			if result == nil {
				t.Error("Expected result with error info, got nil")
			}

			if result != nil && result.Error == "" {
				t.Error("Expected error message in result, got empty string")
			}

			// Verify error is of correct type
			if err != nil {
				// Use errors.As to unwrap and check the error type
				var scraperErr *types.ScraperError
				if !errors.As(err, &scraperErr) {
					t.Errorf("Expected ScraperError in error chain, got %T: %v", err, err)
				} else if scraperErr.Type != errorType {
					t.Errorf("Expected error type %s, got %s", errorType, scraperErr.Type)
				}
			}
		})
	}
}
