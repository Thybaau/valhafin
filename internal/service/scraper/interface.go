package scraper

import (
	"valhafin/internal/service/scraper/types"
)

// Re-export types for backward compatibility
type Scraper = types.Scraper
type SyncResult = types.SyncResult
type ScraperError = types.ScraperError

// Re-export error constructors
var (
	NewAuthError       = types.NewAuthError
	NewNetworkError    = types.NewNetworkError
	NewParsingError    = types.NewParsingError
	NewValidationError = types.NewValidationError
)
