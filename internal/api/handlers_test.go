package api

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"
	encryptionsvc "valhafin/internal/service/encryption"
	"valhafin/internal/service/performance"
	"valhafin/internal/service/price"
	"valhafin/internal/service/sync"

	"github.com/gorilla/mux"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

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

	// Clean up all test data
	_, _ = db.Exec("DELETE FROM transactions_traderepublic")
	_, _ = db.Exec("DELETE FROM transactions_binance")
	_, _ = db.Exec("DELETE FROM transactions_boursedirect")
	_, _ = db.Exec("DELETE FROM asset_prices")
	_, _ = db.Exec("DELETE FROM assets")
	_, _ = db.Exec("DELETE FROM accounts")
}

// setupTestHandler creates a test handler with dependencies
func setupTestHandler(t *testing.T) (*Handler, *database.DB) {
	db := setupTestDB(t)
	if db == nil {
		return nil, nil
	}

	// Create encryption service with test key
	key := make([]byte, 32)
	_, err := rand.Read(key)
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

	handler := NewHandler(db, encryptionService, syncService, priceService, performanceService)
	return handler, db
}

// **Propriété 1: Création de compte avec chiffrement**
// **Valide: Exigences 1.1, 1.2, 1.3, 1.5**
//
// Pour toute plateforme supportée (Trade Republic, Binance, Bourse Direct) et tout ensemble
// d'identifiants valides, lorsqu'un compte est créé, le système doit stocker les identifiants
// de manière chiffrée dans la base de données et aucun identifiant ne doit être stocké en clair.
func TestProperty_AccountCreationWithEncryption(t *testing.T) {
	handler, db := setupTestHandler(t)
	if handler == nil {
		return
	}
	defer cleanupTestDB(t, db)
	defer db.Close()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20 // Reduced for integration tests
	properties := gopter.NewProperties(parameters)

	// Generator for valid account creation requests
	genValidAccountRequest := gen.OneConstOf(
		// Trade Republic
		map[string]interface{}{
			"name":     "Test TR Account",
			"platform": "traderepublic",
			"credentials": map[string]interface{}{
				"phone_number": "+33612345678",
				"pin":          "1234",
			},
		},
		// Binance
		map[string]interface{}{
			"name":     "Test Binance Account",
			"platform": "binance",
			"credentials": map[string]interface{}{
				"api_key":    "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01",
				"api_secret": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01",
			},
		},
		// Bourse Direct
		map[string]interface{}{
			"name":     "Test BD Account",
			"platform": "boursedirect",
			"credentials": map[string]interface{}{
				"username": "testuser",
				"password": "testpassword123",
			},
		},
	)

	properties.Property("account creation encrypts credentials", prop.ForAll(
		func(reqData map[string]interface{}) bool {
			// Clean up before each test
			cleanupTestDB(t, db)

			// Create request
			reqBody, err := json.Marshal(reqData)
			if err != nil {
				t.Logf("Failed to marshal request: %v", err)
				return false
			}

			req := httptest.NewRequest("POST", "/api/accounts", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call handler
			handler.CreateAccountHandler(w, req)

			// Check response status
			if w.Code != http.StatusCreated {
				t.Logf("Expected status 201, got %d: %s", w.Code, w.Body.String())
				return false
			}

			// Parse response
			var account models.Account
			if err := json.NewDecoder(w.Body).Decode(&account); err != nil {
				t.Logf("Failed to decode response: %v", err)
				return false
			}

			// Verify account was created
			if account.ID == "" {
				t.Logf("Account ID is empty")
				return false
			}

			// Retrieve account from database
			dbAccount, err := db.GetAccountByID(account.ID)
			if err != nil {
				t.Logf("Failed to retrieve account from database: %v", err)
				return false
			}

			// Verify credentials are encrypted (not empty and not plaintext)
			if dbAccount.Credentials == "" {
				t.Logf("Credentials are empty in database")
				return false
			}

			// Verify credentials are not stored in plaintext
			credentialsJSON, _ := json.Marshal(reqData["credentials"])
			if dbAccount.Credentials == string(credentialsJSON) {
				t.Logf("Credentials are stored in plaintext!")
				return false
			}

			// Verify we can decrypt the credentials
			decrypted, err := handler.Encryption.Decrypt(dbAccount.Credentials)
			if err != nil {
				t.Logf("Failed to decrypt credentials: %v", err)
				return false
			}

			// Verify decrypted credentials match original
			var decryptedCreds map[string]interface{}
			if err := json.Unmarshal([]byte(decrypted), &decryptedCreds); err != nil {
				t.Logf("Failed to unmarshal decrypted credentials: %v", err)
				return false
			}

			originalCreds := reqData["credentials"].(map[string]interface{})
			for key, value := range originalCreds {
				if decryptedCreds[key] != value {
					t.Logf("Decrypted credential mismatch for key %s", key)
					return false
				}
			}

			return true
		},
		genValidAccountRequest,
	))

	properties.TestingRun(t)
}

// **Propriété 2: Rejet des identifiants invalides**
// **Valide: Exigences 1.4**
//
// Pour tout ensemble d'identifiants invalides (format incorrect, credentials manquants,
// authentification échouée), le système doit rejeter la création du compte et retourner
// un message d'erreur explicite.
func TestProperty_InvalidCredentialsRejection(t *testing.T) {
	handler, db := setupTestHandler(t)
	if handler == nil {
		return
	}
	defer cleanupTestDB(t, db)
	defer db.Close()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 30
	properties := gopter.NewProperties(parameters)

	// Generator for invalid account creation requests
	genInvalidAccountRequest := gen.OneConstOf(
		// Trade Republic - invalid phone number
		map[string]interface{}{
			"name":     "Test TR Account",
			"platform": "traderepublic",
			"credentials": map[string]interface{}{
				"phone_number": "invalid",
				"pin":          "1234",
			},
		},
		// Trade Republic - invalid PIN (not 4 digits)
		map[string]interface{}{
			"name":     "Test TR Account",
			"platform": "traderepublic",
			"credentials": map[string]interface{}{
				"phone_number": "+33612345678",
				"pin":          "12",
			},
		},
		// Trade Republic - missing phone_number
		map[string]interface{}{
			"name":     "Test TR Account",
			"platform": "traderepublic",
			"credentials": map[string]interface{}{
				"pin": "1234",
			},
		},
		// Binance - invalid API key length
		map[string]interface{}{
			"name":     "Test Binance Account",
			"platform": "binance",
			"credentials": map[string]interface{}{
				"api_key":    "short",
				"api_secret": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01",
			},
		},
		// Binance - missing api_secret
		map[string]interface{}{
			"name":     "Test Binance Account",
			"platform": "binance",
			"credentials": map[string]interface{}{
				"api_key": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01",
			},
		},
		// Bourse Direct - username too short
		map[string]interface{}{
			"name":     "Test BD Account",
			"platform": "boursedirect",
			"credentials": map[string]interface{}{
				"username": "ab",
				"password": "testpassword123",
			},
		},
		// Bourse Direct - password too short
		map[string]interface{}{
			"name":     "Test BD Account",
			"platform": "boursedirect",
			"credentials": map[string]interface{}{
				"username": "testuser",
				"password": "short",
			},
		},
		// Missing credentials
		map[string]interface{}{
			"name":        "Test Account",
			"platform":    "traderepublic",
			"credentials": map[string]interface{}{},
		},
	)

	properties.Property("invalid credentials are rejected with error", prop.ForAll(
		func(reqData map[string]interface{}) bool {
			// Create request
			reqBody, err := json.Marshal(reqData)
			if err != nil {
				t.Logf("Failed to marshal request: %v", err)
				return false
			}

			req := httptest.NewRequest("POST", "/api/accounts", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call handler
			handler.CreateAccountHandler(w, req)

			// Check response status - should be 400 Bad Request
			if w.Code != http.StatusBadRequest {
				t.Logf("Expected status 400, got %d: %s", w.Code, w.Body.String())
				return false
			}

			// Parse error response
			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Logf("Failed to decode error response: %v", err)
				return false
			}

			// Verify error message is not empty
			if errResp.Error.Message == "" {
				t.Logf("Error message is empty")
				return false
			}

			// Verify error code is appropriate
			validCodes := map[string]bool{
				"VALIDATION_ERROR":    true,
				"INVALID_CREDENTIALS": true,
			}
			if !validCodes[errResp.Error.Code] {
				t.Logf("Unexpected error code: %s", errResp.Error.Code)
				return false
			}

			return true
		},
		genInvalidAccountRequest,
	))

	properties.TestingRun(t)
}

// **Propriété 3: Suppression en cascade**
// **Valide: Exigences 1.6, 8.7**
//
// Pour tout compte supprimé, toutes les données associées (identifiants, transactions,
// table dédiée) doivent être supprimées de la base de données et aucune donnée orpheline
// ne doit subsister.
func TestProperty_CascadeDelete(t *testing.T) {
	handler, db := setupTestHandler(t)
	if handler == nil {
		return
	}
	defer cleanupTestDB(t, db)
	defer db.Close()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("deleting account removes all associated data", prop.ForAll(
		func(accountName string) bool {
			// Clean up before each test
			cleanupTestDB(t, db)

			// Create a test account
			account := &models.Account{
				Name:        accountName,
				Platform:    "traderepublic",
				Credentials: "encrypted_test_credentials",
			}

			if err := db.CreateAccount(account); err != nil {
				t.Logf("Failed to create account: %v", err)
				return false
			}

			accountID := account.ID

			// Verify account exists
			_, err := db.GetAccountByID(accountID)
			if err != nil {
				t.Logf("Account not found after creation: %v", err)
				return false
			}

			// Delete account via API
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/accounts/%s", accountID), nil)
			req = mux.SetURLVars(req, map[string]string{"id": accountID})
			w := httptest.NewRecorder()

			handler.DeleteAccountHandler(w, req)

			// Check response status
			if w.Code != http.StatusOK {
				t.Logf("Expected status 200, got %d: %s", w.Code, w.Body.String())
				return false
			}

			// Verify account no longer exists
			_, err = db.GetAccountByID(accountID)
			if err == nil {
				t.Logf("Account still exists after deletion")
				return false
			}
			// The error should indicate the account was not found
			if !strings.Contains(err.Error(), "no rows") {
				t.Logf("Unexpected error after deletion: %v", err)
				return false
			}

			// Verify no orphaned data exists
			// Check transactions tables (they should have ON DELETE CASCADE)
			var count int

			err = db.Get(&count, "SELECT COUNT(*) FROM transactions_traderepublic WHERE account_id = $1", accountID)
			if err != nil {
				t.Logf("Failed to check transactions_traderepublic: %v", err)
				return false
			}
			if count > 0 {
				t.Logf("Found %d orphaned transactions in transactions_traderepublic", count)
				return false
			}

			err = db.Get(&count, "SELECT COUNT(*) FROM transactions_binance WHERE account_id = $1", accountID)
			if err != nil {
				t.Logf("Failed to check transactions_binance: %v", err)
				return false
			}
			if count > 0 {
				t.Logf("Found %d orphaned transactions in transactions_binance", count)
				return false
			}

			err = db.Get(&count, "SELECT COUNT(*) FROM transactions_boursedirect WHERE account_id = $1", accountID)
			if err != nil {
				t.Logf("Failed to check transactions_boursedirect: %v", err)
				return false
			}
			if count > 0 {
				t.Logf("Found %d orphaned transactions in transactions_boursedirect", count)
				return false
			}

			return true
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
	))

	properties.TestingRun(t)
}
