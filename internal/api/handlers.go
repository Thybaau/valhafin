package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"valhafin/internal/domain/models"
	"valhafin/internal/repository/database"
	encryptionsvc "valhafin/internal/service/encryption"

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
	DB         *database.DB
	Encryption *encryptionsvc.EncryptionService
	Validator  *CredentialsValidator
}

// NewHandler creates a new Handler with dependencies
func NewHandler(db *database.DB, encryptionService *encryptionsvc.EncryptionService) *Handler {
	return &Handler{
		DB:         db,
		Encryption: encryptionService,
		Validator:  NewCredentialsValidator(),
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
		if err == sql.ErrNoRows {
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
		if err == sql.ErrNoRows {
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
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

// Transaction handlers (stubs for now)
func (h *Handler) GetAccountTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []interface{}{})
}

func (h *Handler) GetAllTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []interface{}{})
}

func (h *Handler) ImportCSVHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

// Performance handlers (stubs for now)
func (h *Handler) GetAccountPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

func (h *Handler) GetGlobalPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

func (h *Handler) GetAssetPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

// Fees handlers (stubs for now)
func (h *Handler) GetAccountFeesHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

func (h *Handler) GetGlobalFeesHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

// Asset handlers (stubs for now)
func (h *Handler) GetAssetPriceHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

func (h *Handler) GetAssetPriceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}
