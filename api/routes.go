package api

import (
	"github.com/gorilla/mux"
)

// SetupRoutes configures all API routes
func SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Apply middleware
	router.Use(CORSMiddleware)
	router.Use(LoggingMiddleware)

	// API routes
	api := router.PathPrefix("/api").Subrouter()

	// Health check
	router.HandleFunc("/health", HealthCheckHandler).Methods("GET")

	// Account routes
	api.HandleFunc("/accounts", GetAccountsHandler).Methods("GET")
	api.HandleFunc("/accounts", CreateAccountHandler).Methods("POST")
	api.HandleFunc("/accounts/{id}", GetAccountHandler).Methods("GET")
	api.HandleFunc("/accounts/{id}", DeleteAccountHandler).Methods("DELETE")
	api.HandleFunc("/accounts/{id}/sync", SyncAccountHandler).Methods("POST")

	// Transaction routes
	api.HandleFunc("/accounts/{id}/transactions", GetAccountTransactionsHandler).Methods("GET")
	api.HandleFunc("/transactions", GetAllTransactionsHandler).Methods("GET")
	api.HandleFunc("/transactions/import", ImportCSVHandler).Methods("POST")

	// Performance routes
	api.HandleFunc("/accounts/{id}/performance", GetAccountPerformanceHandler).Methods("GET")
	api.HandleFunc("/performance", GetGlobalPerformanceHandler).Methods("GET")
	api.HandleFunc("/assets/{isin}/performance", GetAssetPerformanceHandler).Methods("GET")

	// Fees routes
	api.HandleFunc("/accounts/{id}/fees", GetAccountFeesHandler).Methods("GET")
	api.HandleFunc("/fees", GetGlobalFeesHandler).Methods("GET")

	// Asset routes
	api.HandleFunc("/assets/{isin}/price", GetAssetPriceHandler).Methods("GET")
	api.HandleFunc("/assets/{isin}/history", GetAssetPriceHistoryHandler).Methods("GET")

	return router
}
