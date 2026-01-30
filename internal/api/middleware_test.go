package api

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
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

	"github.com/gorilla/mux"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Propriété 18: Validation des entrées API**
// **Valide: Exigences 7.3, 7.4**
//
// Pour toute requête API avec des données invalides (format incorrect, champs manquants,
// valeurs hors limites), le système doit rejeter la requête avec un code HTTP 400 et un
// message d'erreur structuré en JSON décrivant l'erreur.
func TestProperty_APIInputValidation(t *testing.T) {
	handler, db := setupTestHandler(t)
	if handler == nil {
		return
	}
	defer cleanupTestDB(t, db)
	defer db.Close()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	// Generator for invalid account creation requests
	genInvalidRequest := gen.OneConstOf(
		// Missing name
		map[string]interface{}{
			"platform": "traderepublic",
			"credentials": map[string]interface{}{
				"phone_number": "+33612345678",
				"pin":          "1234",
			},
		},
		// Missing platform
		map[string]interface{}{
			"name": "Test Account",
			"credentials": map[string]interface{}{
				"phone_number": "+33612345678",
				"pin":          "1234",
			},
		},
		// Missing credentials
		map[string]interface{}{
			"name":     "Test Account",
			"platform": "traderepublic",
		},
		// Empty credentials
		map[string]interface{}{
			"name":        "Test Account",
			"platform":    "traderepublic",
			"credentials": map[string]interface{}{},
		},
		// Invalid platform
		map[string]interface{}{
			"name":     "Test Account",
			"platform": "invalid_platform",
			"credentials": map[string]interface{}{
				"phone_number": "+33612345678",
				"pin":          "1234",
			},
		},
		// Invalid Trade Republic credentials (missing phone)
		map[string]interface{}{
			"name":     "Test Account",
			"platform": "traderepublic",
			"credentials": map[string]interface{}{
				"pin": "1234",
			},
		},
		// Invalid Trade Republic credentials (invalid PIN format)
		map[string]interface{}{
			"name":     "Test Account",
			"platform": "traderepublic",
			"credentials": map[string]interface{}{
				"phone_number": "+33612345678",
				"pin":          "12", // Too short
			},
		},
		// Invalid Binance credentials (missing api_key)
		map[string]interface{}{
			"name":     "Test Account",
			"platform": "binance",
			"credentials": map[string]interface{}{
				"api_secret": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01",
			},
		},
		// Invalid Binance credentials (api_key too short)
		map[string]interface{}{
			"name":     "Test Account",
			"platform": "binance",
			"credentials": map[string]interface{}{
				"api_key":    "short",
				"api_secret": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01",
			},
		},
		// Invalid Bourse Direct credentials (missing username)
		map[string]interface{}{
			"name":     "Test Account",
			"platform": "boursedirect",
			"credentials": map[string]interface{}{
				"password": "testpassword123",
			},
		},
	)

	properties.Property("invalid API requests return 400 with structured error", prop.ForAll(
		func(invalidReq map[string]interface{}) bool {
			// Create request
			body, _ := json.Marshal(invalidReq)
			req := httptest.NewRequest("POST", "/api/accounts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handler.CreateAccountHandler(rr, req)

			// Verify status code is 400
			if rr.Code != http.StatusBadRequest {
				t.Logf("Expected 400, got %d for request: %+v", rr.Code, invalidReq)
				return false
			}

			// Verify response is valid JSON
			var errorResp ErrorResponse
			if err := json.NewDecoder(rr.Body).Decode(&errorResp); err != nil {
				t.Logf("Response is not valid JSON: %v", err)
				return false
			}

			// Verify error structure
			if errorResp.Error.Code == "" {
				t.Logf("Error code is empty")
				return false
			}

			if errorResp.Error.Message == "" {
				t.Logf("Error message is empty")
				return false
			}

			return true
		},
		genInvalidRequest,
	))

	properties.TestingRun(t)
}

// **Propriété 23: Logging des requêtes et erreurs**
// **Valide: Exigences 7.5**
//
// Pour toute requête API reçue et toute erreur survenue, le système doit créer une entrée
// de log avec timestamp, endpoint, paramètres (sanitisés), et détails de l'erreur le cas échéant.
func TestProperty_RequestAndErrorLogging(t *testing.T) {
	handler, db := setupTestHandler(t)
	if handler == nil {
		return
	}
	defer cleanupTestDB(t, db)
	defer db.Close()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 30
	properties := gopter.NewProperties(parameters)

	// Generator for various HTTP methods and endpoints
	genRequest := gen.OneConstOf(
		struct {
			method   string
			endpoint string
			body     string
		}{"GET", "/api/accounts", ""},
		struct {
			method   string
			endpoint string
			body     string
		}{"POST", "/api/accounts", `{"name":"Test","platform":"traderepublic","credentials":{"phone_number":"+33612345678","pin":"1234"}}`},
		struct {
			method   string
			endpoint string
			body     string
		}{"GET", "/api/transactions", ""},
		struct {
			method   string
			endpoint string
			body     string
		}{"GET", "/health", ""},
	)

	properties.Property("all requests are logged with method, endpoint, status, and duration", prop.ForAll(
		func(reqData struct {
			method   string
			endpoint string
			body     string
		}) bool {
			// Capture log output
			var logBuf bytes.Buffer
			log.SetOutput(&logBuf)
			defer log.SetOutput(os.Stderr)

			// Create router with middleware
			router := mux.NewRouter()
			router.Use(LoggingMiddleware)

			// Add a test handler
			router.HandleFunc(reqData.endpoint, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Create request
			var body io.Reader
			if reqData.body != "" {
				body = strings.NewReader(reqData.body)
			}
			req := httptest.NewRequest(reqData.method, reqData.endpoint, body)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Serve request
			router.ServeHTTP(rr, req)

			// Check that log was created
			logOutput := logBuf.String()
			if logOutput == "" {
				t.Logf("No log output for request: %s %s", reqData.method, reqData.endpoint)
				return false
			}

			// Verify log contains method
			if !strings.Contains(logOutput, reqData.method) {
				t.Logf("Log does not contain method: %s", reqData.method)
				return false
			}

			// Verify log contains endpoint
			if !strings.Contains(logOutput, reqData.endpoint) {
				t.Logf("Log does not contain endpoint: %s", reqData.endpoint)
				return false
			}

			// Verify log contains status code
			if !strings.Contains(logOutput, "200") {
				t.Logf("Log does not contain status code")
				return false
			}

			// Verify log contains duration (should have time units like "ms", "µs", "s")
			hasDuration := strings.Contains(logOutput, "ms") ||
				strings.Contains(logOutput, "µs") ||
				strings.Contains(logOutput, "s") ||
				strings.Contains(logOutput, "ns")

			if !hasDuration {
				t.Logf("Log does not contain duration")
				return false
			}

			return true
		},
		genRequest,
	))

	properties.TestingRun(t)
}

// **Propriété 26: Health check**
// **Valide: Exigences 11.6**
//
// Pour toute requête au endpoint /health, le système doit retourner un statut 200 avec des
// informations sur l'état de l'application (version, uptime, état de la base de données) si
// tout fonctionne, ou 503 si un service critique est indisponible.
func TestProperty_HealthCheck(t *testing.T) {
	handler, db := setupTestHandler(t)
	if handler == nil {
		return
	}
	defer cleanupTestDB(t, db)
	defer db.Close()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20
	properties := gopter.NewProperties(parameters)

	// Generator for version strings (simple alphanumeric strings)
	genVersion := gen.Identifier().Map(func(s string) string {
		if s == "" {
			return "v1.0.0"
		}
		return s
	})

	// Generator for start times (in the past)
	genStartTime := gen.Int64Range(1, 3600).Map(func(seconds int64) time.Time {
		return time.Now().Add(-time.Duration(seconds) * time.Second)
	})

	properties.Property("health check returns 200 with status info when database is up", prop.ForAll(
		func(version string, startTime time.Time) bool {
			// Set handler version and start time
			handler.Version = version
			handler.StartTime = startTime

			// Create request
			req := httptest.NewRequest("GET", "/health", nil)
			rr := httptest.NewRecorder()

			// Call handler
			handler.HealthCheckHandler(rr, req)

			// Verify status code is 200
			if rr.Code != http.StatusOK {
				t.Logf("Expected 200, got %d", rr.Code)
				return false
			}

			// Parse response
			var response map[string]interface{}
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Logf("Failed to decode response: %v", err)
				return false
			}

			// Verify status is "healthy"
			status, ok := response["status"].(string)
			if !ok || status != "healthy" {
				t.Logf("Status is not 'healthy': %v", response["status"])
				return false
			}

			// Verify version is present
			respVersion, ok := response["version"].(string)
			if !ok || respVersion != version {
				t.Logf("Version mismatch: expected %s, got %v", version, response["version"])
				return false
			}

			// Verify uptime is present and is a string
			uptime, ok := response["uptime"].(string)
			if !ok || uptime == "" {
				t.Logf("Uptime is missing or invalid: %v", response["uptime"])
				return false
			}

			// Verify database status is "up"
			dbStatus, ok := response["database"].(string)
			if !ok || dbStatus != "up" {
				t.Logf("Database status is not 'up': %v", response["database"])
				return false
			}

			return true
		},
		genVersion,
		genStartTime,
	))

	properties.TestingRun(t)
}

// Test that health check returns 503 when database is down
func TestHealthCheck_DatabaseDown(t *testing.T) {
	// Create a handler with a closed database connection
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
		return
	}

	// Close the database connection to simulate failure
	db.Close()

	// Create encryption service
	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate encryption key: %v", err)
	}

	encryptionService, err := encryptionsvc.NewEncryptionService(key)
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	// Create services
	scraperFactory := sync.NewScraperFactory()
	syncService := sync.NewService(db, scraperFactory, encryptionService)
	priceService := price.NewYahooFinanceService(db)
	performanceService := performance.NewPerformanceService(db, priceService)
	feesService := fees.NewFeesService(db)

	handler := NewHandler(db, encryptionService, syncService, priceService, performanceService, feesService)
	handler.Version = "test"
	handler.StartTime = time.Now()

	// Create request
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	// Call handler
	handler.HealthCheckHandler(rr, req)

	// Verify status code is 503
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", rr.Code)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify status is "unhealthy"
	status, ok := response["status"].(string)
	if !ok || status != "unhealthy" {
		t.Errorf("Status is not 'unhealthy': %v", response["status"])
	}

	// Verify database status is "down"
	dbStatus, ok := response["database"].(string)
	if !ok || dbStatus != "down" {
		t.Errorf("Database status is not 'down': %v", response["database"])
	}
}

// Test recovery middleware handles panics
func TestRecoveryMiddleware_HandlesPanic(t *testing.T) {
	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	handler := RecoveryMiddleware(panicHandler)

	// Create request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Call handler (should not panic)
	handler.ServeHTTP(rr, req)

	// Verify status code is 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", rr.Code)
	}

	// Parse response
	var response ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify error structure
	if response.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("Expected error code 'INTERNAL_ERROR', got '%s'", response.Error.Code)
	}

	if response.Error.Message == "" {
		t.Errorf("Error message is empty")
	}
}

// Test CORS middleware sets correct headers
func TestCORSMiddleware_SetsHeaders(t *testing.T) {
	// Create a simple handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with CORS middleware
	handler := CORSMiddleware(testHandler)

	// Test regular request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify CORS headers
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Access-Control-Allow-Origin header not set correctly")
	}

	if !strings.Contains(rr.Header().Get("Access-Control-Allow-Methods"), "GET") {
		t.Errorf("Access-Control-Allow-Methods header not set correctly")
	}

	// Test OPTIONS request (preflight)
	req = httptest.NewRequest("OPTIONS", "/test", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify status code is 204
	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected 204 for OPTIONS request, got %d", rr.Code)
	}
}

// Test that invalid date formats return 400
func TestProperty_DateValidation(t *testing.T) {
	handler, db := setupTestHandler(t)
	if handler == nil {
		return
	}
	defer cleanupTestDB(t, db)
	defer db.Close()

	// Create a test account
	account := &models.Account{
		Name:        "Test Account",
		Platform:    "traderepublic",
		Credentials: "encrypted",
	}
	if err := db.CreateAccount(account); err != nil {
		t.Fatalf("Failed to create test account: %v", err)
	}

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20
	properties := gopter.NewProperties(parameters)

	// Generator for invalid date formats
	genInvalidDate := gen.OneConstOf(
		"2024-13-01",  // Invalid month
		"2024-01-32",  // Invalid day
		"2024/01/01",  // Wrong separator
		"01-01-2024",  // Wrong order
		"2024-1-1",    // Missing leading zeros
		"not-a-date",  // Not a date
		"2024-01-01T", // Incomplete
	)

	properties.Property("invalid date formats return 400", prop.ForAll(
		func(invalidDate string) bool {
			// Create request with invalid date
			url := fmt.Sprintf("/api/accounts/%s/fees?start_date=%s", account.ID, invalidDate)
			req := httptest.NewRequest("GET", url, nil)
			req = mux.SetURLVars(req, map[string]string{"id": account.ID})

			rr := httptest.NewRecorder()

			// Call handler
			handler.GetAccountFeesHandler(rr, req)

			// Should return 400 for invalid dates
			if rr.Code != http.StatusBadRequest {
				t.Logf("Expected 400 for invalid date '%s', got %d", invalidDate, rr.Code)
				return false
			}

			// Verify error response
			var errorResp ErrorResponse
			if err := json.NewDecoder(rr.Body).Decode(&errorResp); err != nil {
				t.Logf("Response is not valid JSON: %v", err)
				return false
			}

			if errorResp.Error.Code != "INVALID_DATE" {
				t.Logf("Expected error code 'INVALID_DATE', got '%s'", errorResp.Error.Code)
				return false
			}

			return true
		},
		genInvalidDate,
	))

	properties.TestingRun(t)
}
