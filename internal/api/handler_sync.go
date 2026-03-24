package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"valhafin/internal/service/scraper/traderepublic"

	"github.com/gorilla/mux"
)

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

// CompleteSyncRequest represents the request to complete sync with 2FA code
type CompleteSyncRequest struct {
	ProcessID string `json:"process_id"`
	Code      string `json:"code"`
}

// SyncAccountHandler triggers synchronization for an account
// @Summary Synchroniser un compte
// @Description Déclenche la synchronisation des transactions pour un compte (Binance, Bourse Direct)
// @Tags sync
// @Produce json
// @Param id path string true "ID du compte"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts/{id}/sync [post]
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

// InitSyncHandler initiates synchronization for Trade Republic (triggers 2FA)
// @Summary Initier la synchronisation Trade Republic
// @Description Déclenche l'authentification 2FA pour Trade Republic
// @Tags sync
// @Produce json
// @Param id path string true "ID du compte"
// @Success 200 {object} InitSyncResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts/{id}/sync/init [post]
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
			log.Printf("[SYNC] InitSync failed for account %s: %s", accountID, errMsg)
			respondError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", errMsg, nil)
			return
		}

		log.Printf("[SYNC] InitSync failed for account %s: %s", accountID, authErr.Error())
		respondError(w, http.StatusInternalServerError, "AUTH_ERROR", authErr.Error(), nil)
		return
	}

	// If we get here, authentication succeeded without 2FA (shouldn't happen for TR)
	respondJSON(w, http.StatusOK, InitSyncResponse{
		RequiresTwoFactor: false,
		Message:           "Authentication successful",
	})
}

// CompleteSyncHandler completes synchronization with 2FA code
// @Summary Compléter la synchronisation Trade Republic avec le code 2FA
// @Description Finalise la synchronisation en fournissant le code de vérification
// @Tags sync
// @Accept json
// @Produce json
// @Param id path string true "ID du compte"
// @Param body body CompleteSyncRequest true "Process ID et code 2FA"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts/{id}/sync/complete [post]
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
