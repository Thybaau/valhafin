package api

import (
	"valhafin/internal/repository/database"
	encryptionsvc "valhafin/internal/service/encryption"

	"github.com/gorilla/mux"
)

// SetupRoutes configures all API routes
func SetupRoutes(db *database.DB, encryptionService *encryptionsvc.EncryptionService) *mux.Router {
	router := mux.NewRouter()

	// Create handler with dependencies
	handler := NewHandler(db, encryptionService)

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

	return router
}
