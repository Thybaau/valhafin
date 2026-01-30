package api

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status":   "unhealthy",
			"database": "down",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
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
			continue
		}
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

	transaction.ISIN = getColumn("isin")
	if transaction.ISIN == "" {
		return nil, fmt.Errorf("isin is required")
	}

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
		transaction.ID = fmt.Sprintf("%s_%s_%.2f", transaction.Timestamp, transaction.ISIN, transaction.AmountValue)
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
			transaction.Metadata = string(metadataJSON)
		} else {
			transaction.Metadata = metadata
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
