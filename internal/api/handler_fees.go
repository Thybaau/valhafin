package api

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// GetAccountFeesHandler retrieves fee metrics for a specific account
// @Summary Frais d'un compte
// @Description Calcule les métriques de frais pour un compte spécifique
// @Tags fees
// @Produce json
// @Param id path string true "ID du compte"
// @Param start_date query string false "Date de début (YYYY-MM-DD)"
// @Param end_date query string false "Date de fin (YYYY-MM-DD)"
// @Success 200 {object} fees.FeesMetrics
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts/{id}/fees [get]
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
// @Summary Frais globaux
// @Description Calcule les métriques de frais pour tous les comptes
// @Tags fees
// @Produce json
// @Param start_date query string false "Date de début (YYYY-MM-DD)"
// @Param end_date query string false "Date de fin (YYYY-MM-DD)"
// @Success 200 {object} fees.FeesMetrics
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/fees [get]
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
