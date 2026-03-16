package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"valhafin/internal/domain/models"

	"github.com/gorilla/mux"
)

// CreateAccountRequest represents the request body for creating an account
type CreateAccountRequest struct {
	Name        string                 `json:"name"`
	Platform    string                 `json:"platform"`
	Credentials map[string]interface{} `json:"credentials"`
}

// CreateAccountHandler creates a new account with encrypted credentials
// @Summary Créer un nouveau compte
// @Description Crée un compte financier avec des credentials chiffrés
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body CreateAccountRequest true "Données du compte"
// @Success 201 {object} models.Account
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts [post]
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
// @Summary Lister tous les comptes
// @Description Récupère la liste de tous les comptes financiers
// @Tags accounts
// @Produce json
// @Success 200 {array} models.Account
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts [get]
func (h *Handler) GetAccountsHandler(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.DB.GetAllAccounts()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve accounts", nil)
		return
	}

	respondJSON(w, http.StatusOK, accounts)
}

// GetAccountHandler retrieves a specific account by ID
// @Summary Récupérer un compte par ID
// @Description Retourne les détails d'un compte financier
// @Tags accounts
// @Produce json
// @Param id path string true "ID du compte"
// @Success 200 {object} models.Account
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts/{id} [get]
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
// @Summary Supprimer un compte
// @Description Supprime un compte et toutes ses données associées
// @Tags accounts
// @Produce json
// @Param id path string true "ID du compte"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts/{id} [delete]
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
