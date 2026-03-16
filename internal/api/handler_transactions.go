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

	"github.com/gorilla/mux"
)

// TransactionResponse represents a paginated transaction response
type TransactionResponse struct {
	Transactions []models.Transaction `json:"transactions"`
	Total        int                  `json:"total"`
	Page         int                  `json:"page"`
	Limit        int                  `json:"limit"`
	TotalPages   int                  `json:"total_pages"`
}

// ImportSummary represents the result of a CSV import operation
type ImportSummary struct {
	Imported int      `json:"imported"`
	Ignored  int      `json:"ignored"`
	Errors   int      `json:"errors"`
	Details  []string `json:"details,omitempty"`
}

// GetAccountTransactionsHandler retrieves transactions for a specific account with filters
// @Summary Récupérer les transactions d'un compte
// @Description Retourne les transactions paginées et filtrées d'un compte
// @Tags transactions
// @Produce json
// @Param id path string true "ID du compte"
// @Param start_date query string false "Date de début (YYYY-MM-DD)"
// @Param end_date query string false "Date de fin (YYYY-MM-DD)"
// @Param asset query string false "Filtrer par ISIN"
// @Param type query string false "Filtrer par type (buy, sell, dividend, fee)"
// @Param page query int false "Numéro de page" default(1)
// @Param limit query int false "Nombre de résultats par page" default(50)
// @Param sort_by query string false "Trier par champ (timestamp, amount)"
// @Param sort_order query string false "Ordre de tri (asc, desc)"
// @Success 200 {object} TransactionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts/{id}/transactions [get]
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
// @Summary Récupérer toutes les transactions
// @Description Retourne les transactions paginées de tous les comptes
// @Tags transactions
// @Produce json
// @Param start_date query string false "Date de début (YYYY-MM-DD)"
// @Param end_date query string false "Date de fin (YYYY-MM-DD)"
// @Param asset query string false "Filtrer par ISIN"
// @Param type query string false "Filtrer par type (buy, sell, dividend, fee)"
// @Param page query int false "Numéro de page" default(1)
// @Param limit query int false "Nombre de résultats par page" default(50)
// @Param sort_by query string false "Trier par champ (timestamp, amount)"
// @Param sort_order query string false "Ordre de tri (asc, desc)"
// @Success 200 {object} TransactionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/transactions [get]
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

// UpdateTransactionHandler updates an existing transaction
// @Summary Modifier une transaction
// @Description Met à jour une transaction existante
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "ID de la transaction"
// @Param transaction body models.Transaction true "Données de la transaction"
// @Success 200 {object} models.Transaction
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/transactions/{id} [put]
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
// @Summary Importer des transactions depuis un CSV
// @Description Importe des transactions à partir d'un fichier CSV avec déduplication
// @Tags transactions
// @Accept multipart/form-data
// @Produce json
// @Param account_id formData string true "ID du compte"
// @Param file formData file true "Fichier CSV"
// @Success 200 {object} ImportSummary
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/transactions/import [post]
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
