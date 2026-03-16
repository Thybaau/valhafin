package api

import (
	"encoding/json"
	"net/http"
	"time"
	"valhafin/internal/repository/database"
	encryptionsvc "valhafin/internal/service/encryption"
	"valhafin/internal/service/fees"
	"valhafin/internal/service/performance"
	"valhafin/internal/service/price"
	"valhafin/internal/service/sync"
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
