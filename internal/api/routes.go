package api

import (
	"valhafin/internal/repository/database"
	"valhafin/internal/service/encryption"
	"valhafin/internal/service/price"
	"valhafin/internal/service/sync"

	"github.com/gorilla/mux"
)

// Services holds the application services
type Services struct {
	SyncService  *sync.Service
	PriceService price.Service
}

// SetupRoutes configures all API routes and returns the router and services
func SetupRoutes(db *database.DB, encryptionService *encryption.EncryptionService) (*mux.Router, *Services) {
	router := mux.NewRouter()

	// Create scraper factory
	scraperFactory := sync.NewScraperFactory()

	// Create sync service
	syncService := sync.NewService(db, scraperFactory, encryptionService)

	// Create price service
	priceService := price.NewYahooFinanceService(db)

	// Create handler with dependencies
	handler := NewHandler(db, encryptionService, syncService, priceService)

	// Apply middleware
	router.Use(CORSMiddleware)
	router.Use(LoggingMiddleware)

	// API routes
	api := router.PathPrefix("/api").Subrouter()

	// Health check
	router.HandleFunc("/health", handler.HealthCheckHandler).Methods("GET")

	// Account routes
	api.HandleFunc("/accounts", handler.GetAccountsHandler).Methods("GET")
	api.HandleFunc("/accounts", handler.CreateAccountHandler).Methods("POST")
	api.HandleFunc("/accounts/{id}", handler.GetAccountHandler).Methods("GET")
	api.HandleFunc("/accounts/{id}", handler.DeleteAccountHandler).Methods("DELETE")
	api.HandleFunc("/accounts/{id}/sync", handler.SyncAccountHandler).Methods("POST")

	// Transaction routes
	api.HandleFunc("/accounts/{id}/transactions", handler.GetAccountTransactionsHandler).Methods("GET")
	api.HandleFunc("/transactions", handler.GetAllTransactionsHandler).Methods("GET")
	api.HandleFunc("/transactions/import", handler.ImportCSVHandler).Methods("POST")

	// Performance routes
	api.HandleFunc("/accounts/{id}/performance", handler.GetAccountPerformanceHandler).Methods("GET")
	api.HandleFunc("/performance", handler.GetGlobalPerformanceHandler).Methods("GET")
	api.HandleFunc("/assets/{isin}/performance", handler.GetAssetPerformanceHandler).Methods("GET")

	// Fees routes
	api.HandleFunc("/accounts/{id}/fees", handler.GetAccountFeesHandler).Methods("GET")
	api.HandleFunc("/fees", handler.GetGlobalFeesHandler).Methods("GET")

	// Asset routes
	api.HandleFunc("/assets/{isin}/price", handler.GetAssetPriceHandler).Methods("GET")
	api.HandleFunc("/assets/{isin}/history", handler.GetAssetPriceHistoryHandler).Methods("GET")

	// Return router and services
	services := &Services{
		SyncService:  syncService,
		PriceService: priceService,
	}

	return router, services
}
