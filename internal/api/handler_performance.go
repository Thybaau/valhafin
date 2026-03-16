package api

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// GetAccountPerformanceHandler retrieves performance metrics for a specific account
// @Summary Performance d'un compte
// @Description Calcule les métriques de performance pour un compte spécifique
// @Tags performance
// @Produce json
// @Param id path string true "ID du compte"
// @Param period query string false "Période (1m, 3m, 1y, all)" default(1y)
// @Success 200 {object} performance.Performance
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/accounts/{id}/performance [get]
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
// @Summary Performance globale
// @Description Calcule les métriques de performance pour tous les comptes
// @Tags performance
// @Produce json
// @Param period query string false "Période (1m, 3m, 1y, all)" default(1y)
// @Success 200 {object} performance.Performance
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/performance [get]
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
// @Summary Performance d'un actif
// @Description Calcule les métriques de performance pour un actif spécifique
// @Tags performance
// @Produce json
// @Param isin path string true "Code ISIN de l'actif"
// @Param period query string false "Période (1m, 3m, 1y, all)" default(1y)
// @Success 200 {object} performance.AssetPerformance
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/assets/{isin}/performance [get]
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
