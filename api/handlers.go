package api

import (
	"encoding/json"
	"net/http"
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
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// Account handlers (stubs for now)
func GetAccountsHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []interface{}{})
}

func CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

func GetAccountHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

func DeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

func SyncAccountHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

// Transaction handlers (stubs for now)
func GetAccountTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []interface{}{})
}

func GetAllTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []interface{}{})
}

func ImportCSVHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

// Performance handlers (stubs for now)
func GetAccountPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

func GetGlobalPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

func GetAssetPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

// Fees handlers (stubs for now)
func GetAccountFeesHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

func GetGlobalFeesHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

// Asset handlers (stubs for now)
func GetAssetPriceHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}

func GetAssetPriceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not implemented yet", nil)
}
