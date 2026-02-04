package api

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"
	encryptionsvc "valhafin/internal/service/encryption"
	"valhafin/internal/service/fees"
	"valhafin/internal/service/performance"
	"valhafin/internal/service/price"
	"valhafin/internal/service/scraper/traderepublic"
	"valhafin/internal/service/sync"

	"github.com/gorilla/mux"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Handler holds dependencies for API handlers
type Handler struct {
	DB                 *database.DB
	Encryption         *encryptionsvc.EncryptionService
	Validator          *CredentialsValidator
	SyncService        *sync.Service
	PriceService       price.Service
	PerformanceService performance.Service
	FeesService        fees.Service
	Version            string
	StartTime          time.Time
}

// NewHandler creates a new Handler with dependencies
func NewHandler(db *database.DB, encryptionService *encryptionsvc.EncryptionService, syncService *sync.Service, priceService price.Service, performanceService performance.Service, feesService fees.Service) *Handler {
	return &Handler{
		DB:                 db,
		Encryption:         encryptionService,
		Validator:          NewCredentialsValidator(),
		SyncService:        syncService,
		PriceService:       priceService,
		PerformanceService: performanceService,
		FeesService:        feesService,
		Version:            "dev",
		StartTime:          time.Now(),
	}
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, code, message string, details interface{}) {
	respondJSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// HealthCheckHandler handles health check requests
func (h *Handler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	if err := h.DB.Ping(); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status":   "unhealthy",
			"database": "down",
			"error":    err.Error(),
		})
		return
	}

	uptime := time.Since(h.StartTime)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "healthy",
		"version":  h.Version,
		"uptime":   uptime.String(),
		"database": "up",
	})
}

// CreateAccountRequest represents the request body for creating an account
type CreateAccountRequest struct {
	Name        string                 `json:"name"`
	Platform    string                 `json:"platform"`
	Credentials map[string]interface{} `json:"credentials"`
}

// CreateAccountHandler creates a new account with encrypted credentials
func (h *Handler) CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	// Validate required fields
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Account name is required", map[string]string{
			"field": "name",
		})
		return
	}

	if req.Platform == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Platform is required", map[string]string{
			"field": "platform",
		})
		return
	}

	if req.Credentials == nil || len(req.Credentials) == 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Credentials are required", map[string]string{
			"field": "credentials",
		})
		return
	}

	// Validate platform-specific credentials
	if err := h.Validator.ValidateCredentials(req.Platform, req.Credentials); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", err.Error(), map[string]string{
			"platform": req.Platform,
		})
		return
	}

	// Convert credentials to JSON string
	credentialsJSON, err := json.Marshal(req.Credentials)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process credentials", nil)
		return
	}

	// Encrypt credentials
	encryptedCredentials, err := h.Encryption.Encrypt(string(credentialsJSON))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ENCRYPTION_ERROR", "Failed to encrypt credentials", nil)
		return
	}

	// Create account model
	account := &models.Account{
		Name:        req.Name,
		Platform:    req.Platform,
		Credentials: encryptedCredentials,
	}

	// Save to database
	if err := h.DB.CreateAccount(account); err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create account", nil)
		return
	}

	// Return created account (without credentials)
	respondJSON(w, http.StatusCreated, account)
}

// GetAccountsHandler lists all accounts
func (h *Handler) GetAccountsHandler(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.DB.GetAllAccounts()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve accounts", nil)
		return
	}

	respondJSON(w, http.StatusOK, accounts)
}

// GetAccountHandler retrieves a specific account by ID
func (h *Handler) GetAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["id"]

	if accountID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Account ID is required", nil)
		return
	}

	account, err := h.DB.GetAccountByID(accountID)
	if err != nil {
		if err == sql.ErrNoRows || (err != nil && strings.Contains(err.Error(), "no rows")) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve account", nil)
		return
	}

	respondJSON(w, http.StatusOK, account)
}

// DeleteAccountHandler deletes an account and all associated data (cascade)
func (h *Handler) DeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["id"]

	if accountID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Account ID is required", nil)
		return
	}

	// Check if account exists
	_, err := h.DB.GetAccountByID(accountID)
	if err != nil {
		if err == sql.ErrNoRows || (err != nil && strings.Contains(err.Error(), "no rows")) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve account", nil)
		return
	}

	// Delete account (cascade will handle associated data)
	if err := h.DB.DeleteAccount(accountID); err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete account", nil)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Account deleted successfully",
	})
}

// SyncAccountHandler triggers synchronization for an account
func (h *Handler) SyncAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["id"]

	if accountID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Account ID is required", nil)
		return
	}

	// Check if account exists
	_, err := h.DB.GetAccountByID(accountID)
	if err != nil {
		if err == sql.ErrNoRows || (err != nil && strings.Contains(err.Error(), "no rows")) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve account", nil)
		return
	}

	// Trigger synchronization
	result, err := h.SyncService.SyncAccount(accountID)
	if err != nil {
		// Return the result even if there was an error, as it contains useful information
		if result != nil {
			respondJSON(w, http.StatusOK, result)
			return
		}
		respondError(w, http.StatusInternalServerError, "SYNC_ERROR", "Failed to synchronize account", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// InitSyncRequest represents the request to initiate a sync (for Trade Republic 2FA)
type InitSyncRequest struct {
	Code string `json:"code,omitempty"` // Optional: for completing 2FA
}

// InitSyncResponse represents the response when initiating a sync
type InitSyncResponse struct {
	RequiresTwoFactor bool   `json:"requires_two_factor"`
	ProcessID         string `json:"process_id,omitempty"`
	Message           string `json:"message"`
}

// InitSyncHandler initiates synchronization for Trade Republic (triggers 2FA)
func (h *Handler) InitSyncHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["id"]

	if accountID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Account ID is required", nil)
		return
	}

	// Get account
	account, err := h.DB.GetAccountByID(accountID)
	if err != nil {
		if err == sql.ErrNoRows || (err != nil && strings.Contains(err.Error(), "no rows")) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve account", nil)
		return
	}

	// Only Trade Republic requires 2FA init
	if account.Platform != "traderepublic" {
		respondError(w, http.StatusBadRequest, "INVALID_PLATFORM", "This endpoint is only for Trade Republic accounts", nil)
		return
	}

	// Decrypt credentials
	credentialsJSON, err := h.Encryption.Decrypt(account.Credentials)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DECRYPTION_ERROR", "Failed to decrypt credentials", nil)
		return
	}

	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(credentialsJSON), &credentials); err != nil {
		respondError(w, http.StatusInternalServerError, "PARSING_ERROR", "Failed to parse credentials", nil)
		return
	}

	// Get scraper
	scraper := h.SyncService.GetScraper("traderepublic")
	if scraper == nil {
		respondError(w, http.StatusInternalServerError, "SCRAPER_ERROR", "Trade Republic scraper not available", nil)
		return
	}

	// Cast to Trade Republic scraper to access Authenticate2FA method
	trScraper, ok := scraper.(*traderepublic.Scraper)
	if !ok {
		respondError(w, http.StatusInternalServerError, "SCRAPER_ERROR", "Invalid scraper type", nil)
		return
	}

	// Try to authenticate (this will trigger 2FA and return processID)
	// This will fail with a 2FA required error, but we can extract the processID from the error
	_, authErr := trScraper.FetchTransactions(credentials, nil)

	if authErr != nil {
		// Check if it's a 2FA required error
		errMsg := authErr.Error()
		if strings.Contains(errMsg, "2FA authentication required") && strings.Contains(errMsg, "Process ID:") {
			// Extract process ID from error message
			parts := strings.Split(errMsg, "Process ID: ")
			if len(parts) > 1 {
				processID := strings.TrimSuffix(strings.Split(parts[1], ".")[0], "")

				// Store processID temporarily (in a real app, use Redis or similar)
				// For now, return it to the client
				respondJSON(w, http.StatusOK, InitSyncResponse{
					RequiresTwoFactor: true,
					ProcessID:         processID,
					Message:           "Check your Trade Republic app for the verification code",
				})
				return
			}
		}

		// If it's a login error, it means the credentials are wrong
		if strings.Contains(errMsg, "Login failed") {
			respondError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "Invalid phone number or PIN", map[string]string{
				"error": errMsg,
			})
			return
		}

		respondError(w, http.StatusInternalServerError, "AUTH_ERROR", "Failed to initiate authentication", map[string]string{
			"error": authErr.Error(),
		})
		return
	}

	// If we get here, authentication succeeded without 2FA (shouldn't happen for TR)
	respondJSON(w, http.StatusOK, InitSyncResponse{
		RequiresTwoFactor: false,
		Message:           "Authentication successful",
	})
}

// CompleteSyncRequest represents the request to complete sync with 2FA code
type CompleteSyncRequest struct {
	ProcessID string `json:"process_id"`
	Code      string `json:"code"`
}

// CompleteSyncHandler completes synchronization with 2FA code
func (h *Handler) CompleteSyncHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["id"]

	if accountID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Account ID is required", nil)
		return
	}

	var req CompleteSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.ProcessID == "" || req.Code == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Process ID and code are required", nil)
		return
	}

	// Get account
	account, err := h.DB.GetAccountByID(accountID)
	if err != nil {
		if err == sql.ErrNoRows || (err != nil && strings.Contains(err.Error(), "no rows")) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve account", nil)
		return
	}

	// Only Trade Republic requires 2FA
	if account.Platform != "traderepublic" {
		respondError(w, http.StatusBadRequest, "INVALID_PLATFORM", "This endpoint is only for Trade Republic accounts", nil)
		return
	}

	// Get scraper
	scraper := h.SyncService.GetScraper("traderepublic")
	if scraper == nil {
		respondError(w, http.StatusInternalServerError, "SCRAPER_ERROR", "Trade Republic scraper not available", nil)
		return
	}

	// Cast to Trade Republic scraper
	trScraper, ok := scraper.(*traderepublic.Scraper)
	if !ok {
		respondError(w, http.StatusInternalServerError, "SCRAPER_ERROR", "Invalid scraper type", nil)
		return
	}

	// Complete 2FA authentication
	log.Printf("INFO: Completing 2FA for account %s with process ID %s", accountID, req.ProcessID)
	sessionToken, err := trScraper.Authenticate2FA(req.ProcessID, req.Code)
	if err != nil {
		log.Printf("ERROR: 2FA verification failed for account %s: %v", accountID, err)
		respondError(w, http.StatusBadRequest, "AUTH_ERROR", "Failed to verify code", map[string]string{
			"error": err.Error(),
		})
		return
	}

	if sessionToken == "" {
		log.Printf("ERROR: Empty session token for account %s", accountID)
		respondError(w, http.StatusInternalServerError, "AUTH_ERROR", "Failed to obtain session token", nil)
		return
	}

	log.Printf("INFO: Successfully authenticated, fetching transactions for account %s", accountID)
	// Now fetch transactions using the session token
	// For Trade Republic, always fetch all transactions (don't use lastSync filter)
	// because the WebSocket API returns all transactions anyway
	transactions, err := trScraper.FetchTransactionsWithToken(sessionToken, nil)
	if err != nil {
		log.Printf("ERROR: Failed to fetch transactions for account %s: %v", accountID, err)
		respondError(w, http.StatusInternalServerError, "SYNC_ERROR", "Failed to fetch transactions", map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Printf("INFO: Fetched %d transactions for account %s", len(transactions), accountID)

	// Set account ID for all transactions
	for i := range transactions {
		transactions[i].AccountID = account.ID
	}

	// Store transactions in database
	transactionsStored := 0
	if len(transactions) > 0 {
		if err := h.DB.CreateTransactionsBatch(transactions, account.Platform); err != nil {
			respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to store transactions", map[string]string{
				"error": err.Error(),
			})
			return
		}
		transactionsStored = len(transactions)
	}

	// Resolve symbols for assets with Yahoo Finance
	log.Printf("INFO: Resolving symbols for assets...")
	symbolsResolved := h.resolveAssetSymbols()
	log.Printf("INFO: Resolved %d symbols", symbolsResolved)

	// Update last sync timestamp
	now := time.Now()
	if err := h.DB.UpdateAccountLastSync(account.ID, now); err != nil {
		log.Printf("WARNING: Failed to update last sync timestamp for account %s: %v", account.ID, err)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":            true,
		"transactions_added": transactionsStored,
		"symbols_resolved":   symbolsResolved,
		"message":            fmt.Sprintf("Successfully synchronized %d transactions and resolved %d symbols", transactionsStored, symbolsResolved),
	})
}

// Transaction handlers

// TransactionResponse represents a paginated transaction response
type TransactionResponse struct {
	Transactions []models.Transaction `json:"transactions"`
	Total        int                  `json:"total"`
	Page         int                  `json:"page"`
	Limit        int                  `json:"limit"`
	TotalPages   int                  `json:"total_pages"`
}

// GetAccountTransactionsHandler retrieves transactions for a specific account with filters
func (h *Handler) GetAccountTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["id"]

	if accountID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Account ID is required", nil)
		return
	}

	// Check if account exists and get platform
	account, err := h.DB.GetAccountByID(accountID)
	if err != nil {
		if err == sql.ErrNoRows || (err != nil && strings.Contains(err.Error(), "no rows")) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve account", nil)
		return
	}

	// Parse query parameters
	filter := h.parseTransactionFilters(r)
	filter.AccountID = accountID

	// Get sort parameters
	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")

	// Validate sort parameters
	if sortBy != "" && sortBy != "timestamp" && sortBy != "amount" {
		respondError(w, http.StatusBadRequest, "INVALID_SORT", "sort_by must be 'timestamp' or 'amount'", nil)
		return
	}
	if sortOrder != "" && sortOrder != "asc" && sortOrder != "desc" {
		respondError(w, http.StatusBadRequest, "INVALID_SORT", "sort_order must be 'asc' or 'desc'", nil)
		return
	}

	// Get transactions with filters
	transactions, err := h.DB.GetTransactionsByAccountWithSort(accountID, account.Platform, filter, sortBy, sortOrder)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve transactions", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Get total count for pagination
	total, err := h.DB.CountTransactions(account.Platform, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to count transactions", nil)
		return
	}

	// Calculate total pages
	totalPages := 0
	if filter.Limit > 0 {
		totalPages = (total + filter.Limit - 1) / filter.Limit
	}

	response := TransactionResponse{
		Transactions: transactions,
		Total:        total,
		Page:         filter.Page,
		Limit:        filter.Limit,
		TotalPages:   totalPages,
	}

	respondJSON(w, http.StatusOK, response)
}

// GetAllTransactionsHandler retrieves all transactions across all accounts with filters
func (h *Handler) GetAllTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := h.parseTransactionFilters(r)

	// Get sort parameters
	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")

	// Validate sort parameters
	if sortBy != "" && sortBy != "timestamp" && sortBy != "amount" {
		respondError(w, http.StatusBadRequest, "INVALID_SORT", "sort_by must be 'timestamp' or 'amount'", nil)
		return
	}
	if sortOrder != "" && sortOrder != "asc" && sortOrder != "desc" {
		respondError(w, http.StatusBadRequest, "INVALID_SORT", "sort_order must be 'asc' or 'desc'", nil)
		return
	}

	// Get all accounts to query all platforms
	accounts, err := h.DB.GetAllAccounts()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve accounts", nil)
		return
	}

	// Collect transactions from all platforms
	allTransactions := []models.Transaction{}
	totalCount := 0

	// Get unique platforms
	platforms := make(map[string]bool)
	for _, account := range accounts {
		platforms[account.Platform] = true
	}

	// Query each platform
	for platform := range platforms {
		transactions, err := h.DB.GetAllTransactionsWithSort(platform, filter, sortBy, sortOrder)
		if err != nil {
			// Log error but continue with other platforms
			log.Printf("ERROR: Failed to get transactions for platform %s: %v", platform, err)
			continue
		}
		log.Printf("DEBUG: Found %d transactions for platform %s", len(transactions), platform)
		allTransactions = append(allTransactions, transactions...)

		count, err := h.DB.CountTransactions(platform, filter)
		if err == nil {
			totalCount += count
		}
	}

	// Sort combined results if needed
	if sortBy != "" {
		h.sortTransactions(allTransactions, sortBy, sortOrder)
	}

	// Apply pagination to combined results
	start := 0
	end := len(allTransactions)
	if filter.Limit > 0 && filter.Page > 0 {
		start = (filter.Page - 1) * filter.Limit
		end = start + filter.Limit
		if start > len(allTransactions) {
			start = len(allTransactions)
		}
		if end > len(allTransactions) {
			end = len(allTransactions)
		}
	}

	paginatedTransactions := allTransactions[start:end]

	// Calculate total pages
	totalPages := 0
	if filter.Limit > 0 {
		totalPages = (totalCount + filter.Limit - 1) / filter.Limit
	}

	response := TransactionResponse{
		Transactions: paginatedTransactions,
		Total:        totalCount,
		Page:         filter.Page,
		Limit:        filter.Limit,
		TotalPages:   totalPages,
	}

	respondJSON(w, http.StatusOK, response)
}

// parseTransactionFilters parses query parameters into a TransactionFilter
func (h *Handler) parseTransactionFilters(r *http.Request) database.TransactionFilter {
	filter := database.TransactionFilter{
		StartDate:       r.URL.Query().Get("start_date"),
		EndDate:         r.URL.Query().Get("end_date"),
		ISIN:            r.URL.Query().Get("asset"),
		TransactionType: r.URL.Query().Get("type"),
		Page:            1,
		Limit:           50, // Default limit
	}

	// Parse page
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	return filter
}

// sortTransactions sorts a slice of transactions
func (h *Handler) sortTransactions(transactions []models.Transaction, sortBy, sortOrder string) {
	if sortBy == "" {
		return
	}

	sort.Slice(transactions, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "timestamp":
			less = transactions[i].Timestamp < transactions[j].Timestamp
		case "amount":
			less = transactions[i].AmountValue < transactions[j].AmountValue
		default:
			return false
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

// ImportSummary represents the result of a CSV import operation
type ImportSummary struct {
	Imported int      `json:"imported"`
	Ignored  int      `json:"ignored"`
	Errors   int      `json:"errors"`
	Details  []string `json:"details,omitempty"`
}

// UpdateTransactionHandler updates an existing transaction
func (h *Handler) UpdateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transactionID := vars["id"]

	if transactionID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Transaction ID is required", nil)
		return
	}

	// Parse request body
	var transaction models.Transaction
	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	// Set the ID from URL
	transaction.ID = transactionID

	// Get account to determine platform
	account, err := h.DB.GetAccountByID(transaction.AccountID)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil)
		return
	}

	// Update transaction
	if err := h.DB.UpdateTransaction(&transaction, account.Platform); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Transaction not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update transaction", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, transaction)
}

// ImportCSVHandler imports transactions from a CSV file
func (h *Handler) ImportCSVHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Failed to parse form data", nil)
		return
	}

	// Get account_id from form
	accountID := r.FormValue("account_id")
	if accountID == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "account_id is required", map[string]string{
			"field": "account_id",
		})
		return
	}

	// Check if account exists and get platform
	account, err := h.DB.GetAccountByID(accountID)
	if err != nil {
		if err == sql.ErrNoRows || (err != nil && strings.Contains(err.Error(), "no rows")) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve account", nil)
		return
	}

	// Get the file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "CSV file is required", map[string]string{
			"field": "file",
		})
		return
	}
	defer file.Close()

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		respondError(w, http.StatusBadRequest, "INVALID_FILE", "File must be a CSV file", map[string]string{
			"filename": header.Filename,
		})
		return
	}

	// Parse CSV
	transactions, errors := h.parseCSV(file, accountID)

	// If there are critical parsing errors and no transactions, reject the import
	if len(transactions) == 0 && len(errors) > 0 {
		respondError(w, http.StatusBadRequest, "CSV_PARSE_ERROR", "Failed to parse CSV file", map[string]interface{}{
			"errors": errors,
		})
		return
	}

	// Import transactions with deduplication
	imported := 0
	ignored := 0
	importErrors := []string{}

	// Get existing transaction IDs to detect duplicates
	existingIDs := make(map[string]bool)
	existingTransactions, err := h.DB.GetTransactionsByAccount(accountID, account.Platform, database.TransactionFilter{
		AccountID: accountID,
		Limit:     10000, // Get all existing transactions
	})
	if err == nil {
		for _, t := range existingTransactions {
			existingIDs[t.ID] = true
		}
	}

	for _, transaction := range transactions {
		// Validate transaction
		if err := transaction.Validate(); err != nil {
			importErrors = append(importErrors, fmt.Sprintf("Transaction %s: %s", transaction.ID, err.Error()))
			continue
		}

		// Check if transaction already exists
		if existingIDs[transaction.ID] {
			ignored++
			continue
		}

		// Try to create transaction
		err := h.DB.CreateTransaction(&transaction, account.Platform)
		if err != nil {
			importErrors = append(importErrors, fmt.Sprintf("Transaction %s: %s", transaction.ID, err.Error()))
		} else {
			imported++
			existingIDs[transaction.ID] = true // Mark as existing for subsequent duplicates in same import
		}
	}

	// Combine all errors
	allErrors := append(errors, importErrors...)

	// Create summary
	summary := ImportSummary{
		Imported: imported,
		Ignored:  ignored,
		Errors:   len(allErrors),
		Details:  allErrors,
	}

	respondJSON(w, http.StatusOK, summary)
}

// parseCSV parses a CSV file and returns transactions and errors
func (h *Handler) parseCSV(file io.Reader, accountID string) ([]models.Transaction, []string) {
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, []string{fmt.Sprintf("Failed to read CSV header: %s", err.Error())}
	}

	// Validate required columns
	requiredColumns := []string{"timestamp", "isin", "amount_value", "fees"}
	columnIndices := make(map[string]int)
	errors := []string{}

	for _, required := range requiredColumns {
		found := false
		for i, col := range header {
			if strings.TrimSpace(strings.ToLower(col)) == required {
				columnIndices[required] = i
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Sprintf("Required column '%s' not found in CSV", required))
		}
	}

	// If required columns are missing, return error
	if len(errors) > 0 {
		return nil, errors
	}

	// Map all columns for flexible parsing
	allColumnIndices := make(map[string]int)
	for i, col := range header {
		allColumnIndices[strings.TrimSpace(strings.ToLower(col))] = i
	}

	// Parse rows
	transactions := []models.Transaction{}
	rowNum := 1 // Start at 1 (header is row 0)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("Row %d: Failed to read row: %s", rowNum, err.Error()))
			rowNum++
			continue
		}

		rowNum++

		// Parse transaction from row
		transaction, err := h.parseCSVRow(row, allColumnIndices, accountID, rowNum)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Row %d: %s", rowNum, err.Error()))
			continue
		}

		transactions = append(transactions, *transaction)
	}

	return transactions, errors
}

// parseCSVRow parses a single CSV row into a Transaction
func (h *Handler) parseCSVRow(row []string, columnIndices map[string]int, accountID string, rowNum int) (*models.Transaction, error) {
	transaction := &models.Transaction{
		AccountID: accountID,
	}

	// Helper function to get column value safely
	getColumn := func(name string) string {
		if idx, ok := columnIndices[name]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	// Parse required fields
	transaction.Timestamp = getColumn("timestamp")
	if transaction.Timestamp == "" {
		return nil, fmt.Errorf("timestamp is required")
	}

	// Validate timestamp format (should be RFC3339 or similar)
	_, err := time.Parse(time.RFC3339, transaction.Timestamp)
	if err != nil {
		// Try alternative format
		_, err = time.Parse("2006-01-02T15:04:05", transaction.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp format (expected RFC3339): %s", transaction.Timestamp)
		}
	}

	isinStr := getColumn("isin")
	if isinStr == "" {
		return nil, fmt.Errorf("isin is required")
	}
	transaction.ISIN = &isinStr

	// Parse amount_value
	amountStr := getColumn("amount_value")
	if amountStr == "" {
		return nil, fmt.Errorf("amount_value is required")
	}
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount_value: %s", amountStr)
	}
	transaction.AmountValue = amount

	// Parse fees (required but can be 0)
	feesStr := getColumn("fees")
	if feesStr == "" {
		feesStr = "0"
	}
	transaction.Fees = feesStr

	// Parse optional fields
	transaction.ID = getColumn("id")
	if transaction.ID == "" {
		// Generate ID from timestamp + isin + amount if not provided
		transaction.ID = fmt.Sprintf("%s_%s_%.2f", transaction.Timestamp, isinStr, transaction.AmountValue)
	}

	transaction.Title = getColumn("title")
	transaction.Icon = getColumn("icon")
	transaction.Avatar = getColumn("avatar")
	transaction.Subtitle = getColumn("subtitle")
	transaction.AmountCurrency = getColumn("amount_currency")
	if transaction.AmountCurrency == "" {
		transaction.AmountCurrency = "EUR" // Default currency
	}

	amountFractionStr := getColumn("amount_fraction")
	if amountFractionStr != "" {
		fraction, err := strconv.Atoi(amountFractionStr)
		if err == nil {
			transaction.AmountFraction = fraction
		}
	}

	transaction.Status = getColumn("status")
	transaction.ActionType = getColumn("action_type")
	transaction.ActionPayload = getColumn("action_payload")
	transaction.CashAccountNumber = getColumn("cash_account_number")

	hiddenStr := getColumn("hidden")
	transaction.Hidden = hiddenStr == "true" || hiddenStr == "1"

	deletedStr := getColumn("deleted")
	transaction.Deleted = deletedStr == "true" || deletedStr == "1"

	// Parse detail fields
	transaction.Actions = getColumn("actions")
	transaction.DividendPerShare = getColumn("dividend_per_share")
	transaction.Taxes = getColumn("taxes")
	transaction.Total = getColumn("total")
	transaction.Shares = getColumn("shares")
	transaction.SharePrice = getColumn("share_price")
	transaction.Amount = getColumn("amount")

	// Parse quantity
	quantityStr := getColumn("quantity")
	if quantityStr != "" {
		quantity, err := strconv.ParseFloat(quantityStr, 64)
		if err == nil {
			transaction.Quantity = quantity
		}
	}

	transaction.TransactionType = getColumn("transaction_type")

	// Parse metadata - must be valid JSON or empty
	metadata := getColumn("metadata")
	if metadata != "" {
		// Validate it's valid JSON
		var js json.RawMessage
		if err := json.Unmarshal([]byte(metadata), &js); err != nil {
			// If not valid JSON, wrap it as a JSON string
			metadataJSON, _ := json.Marshal(map[string]string{"raw": metadata})
			metadataStr := string(metadataJSON)
			transaction.Metadata = &metadataStr
		} else {
			transaction.Metadata = &metadata
		}
	}

	return transaction, nil
}

// Performance handlers

// GetAccountPerformanceHandler retrieves performance metrics for a specific account
func (h *Handler) GetAccountPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["id"]

	if accountID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Account ID is required", nil)
		return
	}

	// Check if account exists
	_, err := h.DB.GetAccountByID(accountID)
	if err != nil {
		if err == sql.ErrNoRows || (err != nil && strings.Contains(err.Error(), "no rows")) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve account", nil)
		return
	}

	// Get period from query parameter (default: 1y)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "1y"
	}

	// Validate period
	validPeriods := map[string]bool{"1m": true, "3m": true, "1y": true, "all": true}
	if !validPeriods[period] {
		respondError(w, http.StatusBadRequest, "INVALID_PERIOD", "Period must be one of: 1m, 3m, 1y, all", nil)
		return
	}

	// Calculate performance
	performance, err := h.PerformanceService.CalculateAccountPerformance(accountID, period)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PERFORMANCE_ERROR", "Failed to calculate performance", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, performance)
}

// GetGlobalPerformanceHandler retrieves performance metrics across all accounts
func (h *Handler) GetGlobalPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	// Get period from query parameter (default: 1y)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "1y"
	}

	// Validate period
	validPeriods := map[string]bool{"1m": true, "3m": true, "1y": true, "all": true}
	if !validPeriods[period] {
		respondError(w, http.StatusBadRequest, "INVALID_PERIOD", "Period must be one of: 1m, 3m, 1y, all", nil)
		return
	}

	// Calculate global performance
	performance, err := h.PerformanceService.CalculateGlobalPerformance(period)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PERFORMANCE_ERROR", "Failed to calculate global performance", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, performance)
}

// GetAssetPerformanceHandler retrieves performance metrics for a specific asset
func (h *Handler) GetAssetPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isin := vars["isin"]

	if isin == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "ISIN is required", nil)
		return
	}

	// Get period from query parameter (default: 1y)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "1y"
	}

	// Validate period
	validPeriods := map[string]bool{"1m": true, "3m": true, "1y": true, "all": true}
	if !validPeriods[period] {
		respondError(w, http.StatusBadRequest, "INVALID_PERIOD", "Period must be one of: 1m, 3m, 1y, all", nil)
		return
	}

	// Calculate asset performance
	performance, err := h.PerformanceService.CalculateAssetPerformance(isin, period)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Asset not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "PERFORMANCE_ERROR", "Failed to calculate asset performance", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, performance)
}

// Fees handlers

// GetAccountFeesHandler retrieves fee metrics for a specific account
func (h *Handler) GetAccountFeesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["id"]

	if accountID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Account ID is required", nil)
		return
	}

	// Check if account exists
	_, err := h.DB.GetAccountByID(accountID)
	if err != nil {
		if err == sql.ErrNoRows || (err != nil && strings.Contains(err.Error(), "no rows")) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Account not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve account", nil)
		return
	}

	// Parse date filters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Validate date formats if provided
	if startDate != "" {
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid start_date format (use YYYY-MM-DD)", nil)
			return
		}
	}

	if endDate != "" {
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid end_date format (use YYYY-MM-DD)", nil)
			return
		}
	}

	// Calculate fees
	feesMetrics, err := h.FeesService.CalculateAccountFees(accountID, startDate, endDate)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "FEES_ERROR", "Failed to calculate fees", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, feesMetrics)
}

// GetGlobalFeesHandler retrieves fee metrics across all accounts
func (h *Handler) GetGlobalFeesHandler(w http.ResponseWriter, r *http.Request) {
	// Parse date filters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Validate date formats if provided
	if startDate != "" {
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid start_date format (use YYYY-MM-DD)", nil)
			return
		}
	}

	if endDate != "" {
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid end_date format (use YYYY-MM-DD)", nil)
			return
		}
	}

	// Calculate global fees
	feesMetrics, err := h.FeesService.CalculateGlobalFees(startDate, endDate)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "FEES_ERROR", "Failed to calculate global fees", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, feesMetrics)
}

// Asset handlers

// GetAssetPriceHandler retrieves the current price for an asset by ISIN
func (h *Handler) GetAssetPriceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isin := vars["isin"]

	if isin == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "ISIN is required", nil)
		return
	}

	// Get current price from price service
	price, err := h.PriceService.GetCurrentPrice(isin)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Asset not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "PRICE_ERROR", "Failed to retrieve price", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, price)
}

// GetAssetPriceHistoryHandler retrieves historical prices for an asset
func (h *Handler) GetAssetPriceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isin := vars["isin"]

	if isin == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "ISIN is required", nil)
		return
	}

	// Parse query parameters
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	// Default to last 30 days if not specified
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if startDateStr != "" {
		parsed, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid start_date format (use YYYY-MM-DD)", nil)
			return
		}
		startDate = parsed
	}

	if endDateStr != "" {
		parsed, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid end_date format (use YYYY-MM-DD)", nil)
			return
		}
		endDate = parsed
	}

	// Validate date range
	if startDate.After(endDate) {
		respondError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", "start_date must be before end_date", nil)
		return
	}

	// Get price history from price service
	prices, err := h.PriceService.GetPriceHistory(isin, startDate, endDate)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Asset not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "PRICE_ERROR", "Failed to retrieve price history", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, prices)
}

// UpdateSingleAssetPrice forces an update of a single asset price
func (h *Handler) UpdateSingleAssetPrice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isin := vars["isin"]

	if isin == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ISIN", "ISIN is required", nil)
		return
	}

	log.Printf("Updating price for asset %s...", isin)

	// Update price for this asset
	if err := h.PriceService.UpdateAssetPrice(isin); err != nil {
		respondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update price", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Get the updated price
	price, err := h.DB.GetLatestAssetPrice(isin)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve updated price", map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Price updated successfully",
		"price":   price,
	})
}

// AssetPosition represents a user's position in an asset
type AssetPosition struct {
	ISIN              string     `json:"isin"`
	Name              string     `json:"name"`
	Quantity          float64    `json:"quantity"`
	AverageBuyPrice   float64    `json:"average_buy_price"`
	CurrentPrice      float64    `json:"current_price"`
	CurrentValue      float64    `json:"current_value"`
	TotalInvested     float64    `json:"total_invested"`
	UnrealizedGain    float64    `json:"unrealized_gain"`
	UnrealizedGainPct float64    `json:"unrealized_gain_pct"`
	Currency          string     `json:"currency"`
	Purchases         []Purchase `json:"purchases"`
}

// Purchase represents a buy transaction
type Purchase struct {
	Date     string  `json:"date"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

// GetAssetsHandler returns all assets with user positions
func (h *Handler) GetAssetsHandler(w http.ResponseWriter, r *http.Request) {
	// Get all accounts
	accounts, err := h.DB.GetAllAccounts()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get accounts", map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Map to store positions by ISIN
	positionsByISIN := make(map[string]*AssetPosition)

	// Collect all transactions from all accounts
	for _, account := range accounts {
		filter := database.TransactionFilter{}
		transactions, err := h.DB.GetTransactionsByAccount(account.ID, account.Platform, filter)
		if err != nil {
			log.Printf("Warning: failed to get transactions for account %s: %v", account.ID, err)
			continue
		}

		// Process transactions
		for _, tx := range transactions {
			if tx.ISIN == nil || *tx.ISIN == "" {
				continue
			}

			isin := *tx.ISIN

			// Initialize position if not exists
			if _, exists := positionsByISIN[isin]; !exists {
				// Get asset info
				asset, err := h.DB.GetAssetByISIN(isin)
				assetName := "Unknown"
				currency := "EUR"
				if err == nil {
					assetName = asset.Name
					currency = asset.Currency
				}

				positionsByISIN[isin] = &AssetPosition{
					ISIN:      isin,
					Name:      assetName,
					Currency:  currency,
					Purchases: []Purchase{},
				}
			}

			position := positionsByISIN[isin]

			// Process based on transaction type
			switch tx.TransactionType {
			case "buy":
				position.Quantity += tx.Quantity
				investedAmount := -tx.AmountValue // AmountValue is negative for buys
				position.TotalInvested += investedAmount

				// Add to purchases list
				position.Purchases = append(position.Purchases, Purchase{
					Date:     tx.Timestamp[:10], // Extract date part
					Quantity: tx.Quantity,
					Price:    investedAmount / tx.Quantity,
				})

			case "sell":
				position.Quantity -= tx.Quantity
				// Reduce invested amount proportionally
				if position.Quantity > 0 {
					avgCost := position.TotalInvested / (position.Quantity + tx.Quantity)
					position.TotalInvested -= avgCost * tx.Quantity
				} else {
					position.TotalInvested = 0
				}
			}
		}
	}

	// Calculate current values and get current prices
	var assets []AssetPosition
	for _, position := range positionsByISIN {
		if position.Quantity <= 0 {
			continue // Skip sold positions
		}

		// Calculate average buy price
		if position.Quantity > 0 {
			position.AverageBuyPrice = position.TotalInvested / position.Quantity
		}

		// Get current price
		currentPrice, err := h.PriceService.GetCurrentPrice(position.ISIN)
		if err != nil {
			log.Printf("Warning: failed to get current price for %s: %v", position.ISIN, err)
			// Use average buy price as fallback
			position.CurrentPrice = position.AverageBuyPrice
		} else {
			position.CurrentPrice = currentPrice.Price
		}

		// Calculate current value and gains
		position.CurrentValue = position.Quantity * position.CurrentPrice
		position.UnrealizedGain = position.CurrentValue - position.TotalInvested
		if position.TotalInvested > 0 {
			position.UnrealizedGainPct = (position.UnrealizedGain / position.TotalInvested) * 100
		}

		assets = append(assets, *position)
	}

	// Sort by current value (descending)
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].CurrentValue > assets[j].CurrentValue
	})

	respondJSON(w, http.StatusOK, assets)
}

// SymbolSearchHandler searches for symbols on Yahoo Finance
func (h *Handler) SymbolSearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		respondError(w, http.StatusBadRequest, "INVALID_QUERY", "Query parameter is required", nil)
		return
	}

	// Call Yahoo Finance search API
	yahooService, ok := h.PriceService.(*price.YahooFinanceService)
	if !ok {
		respondError(w, http.StatusInternalServerError, "SERVICE_ERROR", "Price service is not Yahoo Finance", nil)
		return
	}

	results, err := yahooService.SearchSymbol(query)
	if err != nil {
		log.Printf("ERROR: Yahoo Finance search failed: %v", err)
		respondError(w, http.StatusBadRequest, "SEARCH_ERROR", err.Error(), nil)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

// resolveAssetSymbols resolves Yahoo Finance symbols for assets that don't have verified symbols
func (h *Handler) resolveAssetSymbols() int {
	yahooService, ok := h.PriceService.(*price.YahooFinanceService)
	if !ok {
		log.Printf("WARNING: Price service is not Yahoo Finance, skipping symbol resolution")
		return 0
	}

	// Get all assets without verified symbols
	query := `
		SELECT isin, name, symbol 
		FROM assets 
		WHERE (symbol_verified = false OR symbol_verified IS NULL)
		AND isin IS NOT NULL
	`

	type AssetInfo struct {
		ISIN   string  `db:"isin"`
		Name   string  `db:"name"`
		Symbol *string `db:"symbol"`
	}

	var assets []AssetInfo
	if err := h.DB.Select(&assets, query); err != nil {
		log.Printf("ERROR: Failed to get assets for symbol resolution: %v", err)
		return 0
	}

	log.Printf("INFO: Found %d assets to resolve symbols for", len(assets))

	resolved := 0
	for _, asset := range assets {
		// Get metadata from transactions to extract exchange info
		var metadata struct {
			Symbol    string   `json:"symbol"`
			Exchanges []string `json:"exchanges"`
			Name      string   `json:"name"`
		}

		// Try to get metadata from a transaction with this ISIN
		metadataQuery := `
			SELECT metadata 
			FROM transactions_traderepublic 
			WHERE isin = $1 AND metadata IS NOT NULL 
			LIMIT 1
		`
		var metadataJSON *string
		err := h.DB.Get(&metadataJSON, metadataQuery, asset.ISIN)
		if err == nil && metadataJSON != nil {
			if err := json.Unmarshal([]byte(*metadataJSON), &metadata); err != nil {
				log.Printf("WARNING: Failed to parse metadata for ISIN %s: %v", asset.ISIN, err)
			}
		}

		// Use symbol from metadata or from asset
		symbolToResolve := metadata.Symbol
		if symbolToResolve == "" && asset.Symbol != nil {
			symbolToResolve = *asset.Symbol
		}

		if symbolToResolve == "" {
			log.Printf("WARNING: No symbol found for ISIN %s, skipping", asset.ISIN)
			continue
		}

		// Use asset name from metadata or database
		assetName := metadata.Name
		if assetName == "" {
			assetName = asset.Name
		}

		// Resolve symbol with Yahoo Finance
		resolvedSymbol, verified, err := yahooService.ResolveSymbolWithExchange(
			symbolToResolve,
			metadata.Exchanges,
			assetName,
		)

		if err != nil {
			log.Printf("WARNING: Failed to resolve symbol for ISIN %s (%s): %v", asset.ISIN, symbolToResolve, err)
			continue
		}

		// Update asset with resolved symbol
		updateQuery := `
			UPDATE assets 
			SET symbol = $1, symbol_verified = $2, last_updated = NOW()
			WHERE isin = $3
		`
		if _, err := h.DB.Exec(updateQuery, resolvedSymbol, verified, asset.ISIN); err != nil {
			log.Printf("ERROR: Failed to update symbol for ISIN %s: %v", asset.ISIN, err)
			continue
		}

		log.Printf("INFO: Resolved symbol for %s: %s → %s (verified: %v)", asset.ISIN, symbolToResolve, resolvedSymbol, verified)
		resolved++

		// Small delay to be respectful to Yahoo Finance
		time.Sleep(200 * time.Millisecond)
	}

	return resolved
}

// ResolveAllSymbolsHandler manually triggers symbol resolution for all assets
func (h *Handler) ResolveAllSymbolsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("INFO: Manual symbol resolution triggered")

	resolved := h.resolveAssetSymbols()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":          true,
		"symbols_resolved": resolved,
		"message":          fmt.Sprintf("Successfully resolved %d symbols", resolved),
	})
}

// UpdateAssetSymbolHandler updates the symbol for an asset
func (h *Handler) UpdateAssetSymbolHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isin := vars["isin"]

	if isin == "" {
		respondError(w, http.StatusBadRequest, "INVALID_ISIN", "ISIN is required", nil)
		return
	}

	// Parse request body
	var req struct {
		Symbol         string `json:"symbol"`
		SymbolVerified bool   `json:"symbol_verified"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error())
		return
	}

	// Update asset symbol in database
	query := `
		UPDATE assets 
		SET symbol = $1, symbol_verified = $2, last_updated = NOW()
		WHERE isin = $3
		RETURNING isin, name, symbol, symbol_verified, type, currency, last_updated
	`

	var asset models.Asset
	err := h.DB.Get(&asset, query, req.Symbol, req.SymbolVerified, isin)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "ASSET_NOT_FOUND", "Asset not found", nil)
			return
		}
		log.Printf("ERROR: Failed to update asset symbol: %v", err)
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update asset symbol", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, asset)
}
