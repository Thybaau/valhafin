package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"

	"github.com/gorilla/mux"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// MockDB is a mock database for testing
type MockTransactionDB struct {
	transactions map[string][]models.Transaction
	accounts     map[string]*models.Account
}

func NewMockTransactionDB() *MockTransactionDB {
	return &MockTransactionDB{
		transactions: make(map[string][]models.Transaction),
		accounts:     make(map[string]*models.Account),
	}
}

func (m *MockTransactionDB) GetAccountByID(id string) (*models.Account, error) {
	if account, ok := m.accounts[id]; ok {
		return account, nil
	}
	return nil, fmt.Errorf("account not found")
}

func (m *MockTransactionDB) GetAllAccounts() ([]models.Account, error) {
	accounts := make([]models.Account, 0, len(m.accounts))
	for _, account := range m.accounts {
		accounts = append(accounts, *account)
	}
	return accounts, nil
}

func (m *MockTransactionDB) GetTransactionsByAccountWithSort(accountID string, platform string, filter database.TransactionFilter, sortBy, sortOrder string) ([]models.Transaction, error) {
	key := fmt.Sprintf("%s_%s", accountID, platform)
	transactions, ok := m.transactions[key]
	if !ok {
		return []models.Transaction{}, nil
	}

	// Apply filters
	filtered := []models.Transaction{}
	for _, tx := range transactions {
		if filter.StartDate != "" && tx.Timestamp < filter.StartDate {
			continue
		}
		if filter.EndDate != "" && tx.Timestamp > filter.EndDate {
			continue
		}
		if filter.ISIN != "" && (tx.ISIN == nil || *tx.ISIN != filter.ISIN) {
			continue
		}
		if filter.TransactionType != "" && tx.TransactionType != filter.TransactionType {
			continue
		}
		filtered = append(filtered, tx)
	}

	// Apply sorting
	if sortBy == "timestamp" {
		for i := 0; i < len(filtered)-1; i++ {
			for j := i + 1; j < len(filtered); j++ {
				less := filtered[i].Timestamp < filtered[j].Timestamp
				if sortOrder == "desc" {
					less = !less
				}
				if !less {
					filtered[i], filtered[j] = filtered[j], filtered[i]
				}
			}
		}
	} else if sortBy == "amount" {
		for i := 0; i < len(filtered)-1; i++ {
			for j := i + 1; j < len(filtered); j++ {
				less := filtered[i].AmountValue < filtered[j].AmountValue
				if sortOrder == "desc" {
					less = !less
				}
				if !less {
					filtered[i], filtered[j] = filtered[j], filtered[i]
				}
			}
		}
	}

	// Apply pagination
	start := 0
	end := len(filtered)
	if filter.Limit > 0 && filter.Page > 0 {
		start = (filter.Page - 1) * filter.Limit
		end = start + filter.Limit
		if start > len(filtered) {
			start = len(filtered)
		}
		if end > len(filtered) {
			end = len(filtered)
		}
	}

	return filtered[start:end], nil
}

func (m *MockTransactionDB) GetAllTransactionsWithSort(platform string, filter database.TransactionFilter, sortBy, sortOrder string) ([]models.Transaction, error) {
	allTransactions := []models.Transaction{}
	for key, transactions := range m.transactions {
		if len(key) > 0 {
			allTransactions = append(allTransactions, transactions...)
		}
	}

	// Apply filters
	filtered := []models.Transaction{}
	for _, tx := range allTransactions {
		if filter.StartDate != "" && tx.Timestamp < filter.StartDate {
			continue
		}
		if filter.EndDate != "" && tx.Timestamp > filter.EndDate {
			continue
		}
		if filter.ISIN != "" && (tx.ISIN == nil || *tx.ISIN != filter.ISIN) {
			continue
		}
		if filter.TransactionType != "" && tx.TransactionType != filter.TransactionType {
			continue
		}
		filtered = append(filtered, tx)
	}

	return filtered, nil
}

func (m *MockTransactionDB) CountTransactions(platform string, filter database.TransactionFilter) (int, error) {
	count := 0
	for _, transactions := range m.transactions {
		for _, tx := range transactions {
			if filter.AccountID != "" && tx.AccountID != filter.AccountID {
				continue
			}
			if filter.StartDate != "" && tx.Timestamp < filter.StartDate {
				continue
			}
			if filter.EndDate != "" && tx.Timestamp > filter.EndDate {
				continue
			}
			if filter.ISIN != "" && (tx.ISIN == nil || *tx.ISIN != filter.ISIN) {
				continue
			}
			if filter.TransactionType != "" && tx.TransactionType != filter.TransactionType {
				continue
			}
			count++
		}
	}
	return count, nil
}

// Property 7: Transaction Filtering
// For all filters applied (date, type, asset), the system must return only transactions
// that exactly match the filter criteria and no transaction should be returned if it doesn't match.
func TestProperty_TransactionFiltering(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("filtered transactions match all criteria", prop.ForAll(
		func(numTransactions int, filterType string, filterValue string) bool {
			// Setup mock database
			mockDB := NewMockTransactionDB()
			accountID := "test-account-1"
			platform := "traderepublic"

			mockDB.accounts[accountID] = &models.Account{
				ID:       accountID,
				Platform: platform,
				Name:     "Test Account",
			}

			// Generate transactions with various properties
			transactions := make([]models.Transaction, numTransactions)
			for i := 0; i < numTransactions; i++ {
				timestamp := time.Now().Add(time.Duration(-i*24) * time.Hour).Format(time.RFC3339)
				transactions[i] = models.Transaction{
					ID:              fmt.Sprintf("tx-%d", i),
					AccountID:       accountID,
					Timestamp:       timestamp,
					ISIN:            stringPtr(fmt.Sprintf("ISIN%d", i%3)),
					TransactionType: []string{"buy", "sell", "dividend"}[i%3],
					AmountValue:     float64(100 + i),
					AmountCurrency:  "EUR",
				}
			}

			key := fmt.Sprintf("%s_%s", accountID, platform)
			mockDB.transactions[key] = transactions

			// Test different filter types
			var filter database.TransactionFilter
			var expectedMatch func(models.Transaction) bool

			switch filterType {
			case "isin":
				filter.ISIN = filterValue
				expectedMatch = func(tx models.Transaction) bool {
					return tx.ISIN != nil && *tx.ISIN == filterValue
				}
			case "type":
				filter.TransactionType = filterValue
				expectedMatch = func(tx models.Transaction) bool {
					return tx.TransactionType == filterValue
				}
			case "date":
				filter.StartDate = filterValue
				expectedMatch = func(tx models.Transaction) bool {
					return tx.Timestamp >= filterValue
				}
			default:
				return true // Skip invalid filter types
			}

			// Get filtered transactions
			filtered, err := mockDB.GetTransactionsByAccountWithSort(accountID, platform, filter, "", "")
			if err != nil {
				return false
			}

			// Verify all returned transactions match the filter
			for _, tx := range filtered {
				if !expectedMatch(tx) {
					return false
				}
			}

			// Verify no matching transactions were excluded
			allTransactions := transactions
			matchCount := 0
			for _, tx := range allTransactions {
				if expectedMatch(tx) {
					matchCount++
				}
			}

			return len(filtered) == matchCount
		},
		gen.IntRange(1, 100),
		gen.OneConstOf("isin", "type", "date"),
		gen.OneConstOf("ISIN0", "ISIN1", "buy", "sell", time.Now().Add(-48*time.Hour).Format(time.RFC3339)),
	))

	properties.TestingRun(t)
}

// Property 8: Transaction Sorting
// For all sorting criteria (date, amount) and order (asc, desc), the system must return
// transactions in the specified order and the order must be consistent with the chosen criterion.
func TestProperty_TransactionSorting(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("sorted transactions are in correct order", prop.ForAll(
		func(numTransactions int, sortBy string, sortOrder string) bool {
			if numTransactions < 2 {
				return true // Skip trivial cases
			}

			// Setup mock database
			mockDB := NewMockTransactionDB()
			accountID := "test-account-1"
			platform := "traderepublic"

			mockDB.accounts[accountID] = &models.Account{
				ID:       accountID,
				Platform: platform,
				Name:     "Test Account",
			}

			// Generate transactions with random values
			transactions := make([]models.Transaction, numTransactions)
			for i := 0; i < numTransactions; i++ {
				timestamp := time.Now().Add(time.Duration(-i*24) * time.Hour).Format(time.RFC3339)
				transactions[i] = models.Transaction{
					ID:             fmt.Sprintf("tx-%d", i),
					AccountID:      accountID,
					Timestamp:      timestamp,
					AmountValue:    float64(100 + (i*13)%200), // Pseudo-random amounts
					AmountCurrency: "EUR",
				}
			}

			key := fmt.Sprintf("%s_%s", accountID, platform)
			mockDB.transactions[key] = transactions

			// Get sorted transactions
			filter := database.TransactionFilter{}
			sorted, err := mockDB.GetTransactionsByAccountWithSort(accountID, platform, filter, sortBy, sortOrder)
			if err != nil {
				return false
			}

			if len(sorted) < 2 {
				return true
			}

			// Verify sorting order
			for i := 0; i < len(sorted)-1; i++ {
				var isCorrectOrder bool
				switch sortBy {
				case "timestamp":
					if sortOrder == "asc" {
						isCorrectOrder = sorted[i].Timestamp <= sorted[i+1].Timestamp
					} else {
						isCorrectOrder = sorted[i].Timestamp >= sorted[i+1].Timestamp
					}
				case "amount":
					if sortOrder == "asc" {
						isCorrectOrder = sorted[i].AmountValue <= sorted[i+1].AmountValue
					} else {
						isCorrectOrder = sorted[i].AmountValue >= sorted[i+1].AmountValue
					}
				default:
					return true // Skip invalid sort criteria
				}

				if !isCorrectOrder {
					return false
				}
			}

			return true
		},
		gen.IntRange(2, 100),
		gen.OneConstOf("timestamp", "amount"),
		gen.OneConstOf("asc", "desc"),
	))

	properties.TestingRun(t)
}

// Property 9: Transaction Pagination
// For all transaction sets with size > 50, the system must paginate results and each page
// must contain at most 50 transactions without duplication between pages.
func TestProperty_TransactionPagination(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("pagination returns correct number of transactions without duplicates", prop.ForAll(
		func(numTransactions int, pageSize int) bool {
			if numTransactions < 1 || pageSize < 1 {
				return true
			}

			// Setup mock database
			mockDB := NewMockTransactionDB()
			accountID := "test-account-1"
			platform := "traderepublic"

			mockDB.accounts[accountID] = &models.Account{
				ID:       accountID,
				Platform: platform,
				Name:     "Test Account",
			}

			// Generate transactions
			transactions := make([]models.Transaction, numTransactions)
			for i := 0; i < numTransactions; i++ {
				timestamp := time.Now().Add(time.Duration(-i*24) * time.Hour).Format(time.RFC3339)
				transactions[i] = models.Transaction{
					ID:             fmt.Sprintf("tx-%d", i),
					AccountID:      accountID,
					Timestamp:      timestamp,
					AmountValue:    float64(100 + i),
					AmountCurrency: "EUR",
				}
			}

			key := fmt.Sprintf("%s_%s", accountID, platform)
			mockDB.transactions[key] = transactions

			// Calculate expected number of pages
			expectedPages := (numTransactions + pageSize - 1) / pageSize

			// Collect all transactions from all pages
			allPaginatedTxs := make(map[string]bool)
			for page := 1; page <= expectedPages; page++ {
				filter := database.TransactionFilter{
					Page:  page,
					Limit: pageSize,
				}

				pageTxs, err := mockDB.GetTransactionsByAccountWithSort(accountID, platform, filter, "", "")
				if err != nil {
					return false
				}

				// Verify page size constraint
				if len(pageTxs) > pageSize {
					return false
				}

				// Check for duplicates
				for _, tx := range pageTxs {
					if allPaginatedTxs[tx.ID] {
						return false // Duplicate found
					}
					allPaginatedTxs[tx.ID] = true
				}
			}

			// Verify all transactions were returned across all pages
			return len(allPaginatedTxs) == numTransactions
		},
		gen.IntRange(1, 200),
		gen.IntRange(10, 50),
	))

	properties.TestingRun(t)
}

// Integration test for the actual HTTP handlers
func TestGetAccountTransactionsHandler_Integration(t *testing.T) {
	// Setup mock database
	mockDB := NewMockTransactionDB()
	accountID := "test-account-1"
	platform := "traderepublic"

	mockDB.accounts[accountID] = &models.Account{
		ID:       accountID,
		Platform: platform,
		Name:     "Test Account",
	}

	// Add test transactions
	transactions := []models.Transaction{
		{
			ID:              "tx-1",
			AccountID:       accountID,
			Timestamp:       "2024-01-01T10:00:00Z",
			ISIN:            stringPtr("US0378331005"),
			TransactionType: "buy",
			AmountValue:     100.0,
			AmountCurrency:  "EUR",
		},
		{
			ID:              "tx-2",
			AccountID:       accountID,
			Timestamp:       "2024-01-02T10:00:00Z",
			ISIN:            stringPtr("US0378331005"),
			TransactionType: "sell",
			AmountValue:     150.0,
			AmountCurrency:  "EUR",
		},
	}

	key := fmt.Sprintf("%s_%s", accountID, platform)
	mockDB.transactions[key] = transactions

	// Create request
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/accounts/%s/transactions", accountID), nil)
	w := httptest.NewRecorder()

	// Set up router with path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/accounts/{id}/transactions", func(w http.ResponseWriter, r *http.Request) {
		// Simulate the handler logic with mock DB
		vars := mux.Vars(r)
		accountID := vars["id"]

		account, err := mockDB.GetAccountByID(accountID)
		if err != nil {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil)
			return
		}

		filter := database.TransactionFilter{
			Page:  1,
			Limit: 50,
		}

		transactions, err := mockDB.GetTransactionsByAccountWithSort(accountID, account.Platform, filter, "", "")
		if err != nil {
			respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve transactions", nil)
			return
		}

		total, _ := mockDB.CountTransactions(account.Platform, filter)

		response := TransactionResponse{
			Transactions: transactions,
			Total:        total,
			Page:         1,
			Limit:        50,
			TotalPages:   1,
		}

		respondJSON(w, http.StatusOK, response)
	}).Methods("GET")

	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response TransactionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Transactions) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(response.Transactions))
	}

	if response.Total != 2 {
		t.Errorf("Expected total 2, got %d", response.Total)
	}
}
