package traderepublic

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/service/scraper/types"
)

const (
	baseURL   = "https://api.traderepublic.com"
	wsURL     = "wss://api.traderepublic.com"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
)

// Scraper implements the scraper.Scraper interface for Trade Republic
type Scraper struct {
	client *http.Client
}

// NewScraper creates a new Trade Republic scraper
func NewScraper() *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetPlatformName returns the platform identifier
func (s *Scraper) GetPlatformName() string {
	return "traderepublic"
}

// ValidateCredentials checks if the provided credentials are valid for Trade Republic
func (s *Scraper) ValidateCredentials(credentials map[string]interface{}) error {
	phoneNumber, ok := credentials["phone_number"].(string)
	if !ok || phoneNumber == "" {
		return types.NewValidationError("traderepublic", "phone_number is required", nil)
	}

	pin, ok := credentials["pin"].(string)
	if !ok || pin == "" {
		return types.NewValidationError("traderepublic", "pin is required", nil)
	}

	// Validate PIN format (should be 4 digits)
	if len(pin) != 4 {
		return types.NewValidationError("traderepublic", "PIN must be 4 digits", nil)
	}

	// Validate phone number format (basic check)
	if len(phoneNumber) < 10 {
		return types.NewValidationError("traderepublic", "Invalid phone number format", nil)
	}

	return nil
}

// FetchTransactions retrieves transactions from Trade Republic
func (s *Scraper) FetchTransactions(credentials map[string]interface{}, lastSync *time.Time) ([]models.Transaction, error) {
	// Validate credentials first
	if err := s.ValidateCredentials(credentials); err != nil {
		return nil, err
	}

	// Extract credentials
	phoneNumber := credentials["phone_number"].(string)
	pin := credentials["pin"].(string)

	// Authenticate and get session token (this will trigger 2FA)
	_, err := s.authenticate(phoneNumber, pin)
	if err != nil {
		// Return the error as-is, it contains the processID for 2FA
		return nil, err
	}

	// If we get here without error, authentication succeeded (shouldn't happen for TR)
	// This is a placeholder - actual transaction fetching would happen here
	return []models.Transaction{}, nil
}

// FetchTransactionsWithToken fetches transactions using an authenticated session token via WebSocket
func (s *Scraper) FetchTransactionsWithToken(sessionToken string, lastSync *time.Time) ([]models.Transaction, error) {
	log.Printf("DEBUG: Connecting to Trade Republic WebSocket...")

	// Create WebSocket client
	wsClient, err := NewWebSocketClient(sessionToken)
	if err != nil {
		return nil, types.NewNetworkError("traderepublic", "Failed to connect to WebSocket", err)
	}
	defer wsClient.Close()

	log.Printf("DEBUG: WebSocket connected, fetching timeline...")

	// Fetch timeline transactions
	timelineTransactions, err := wsClient.FetchTimeline()
	if err != nil {
		return nil, types.NewNetworkError("traderepublic", "Failed to fetch timeline transactions", err)
	}

	log.Printf("DEBUG: Received %d timeline transactions", len(timelineTransactions))

	// Convert timeline transactions to our Transaction model
	transactions := s.convertTimelineTransactions(timelineTransactions, wsClient)

	// Filter by lastSync if provided (incremental sync)
	if lastSync != nil {
		filtered := make([]models.Transaction, 0)
		for _, tx := range transactions {
			// Parse transaction timestamp
			txTime, err := time.Parse(time.RFC3339, tx.Timestamp)
			if err == nil && txTime.After(*lastSync) {
				filtered = append(filtered, tx)
			}
		}
		return filtered, nil
	}

	return transactions, nil
}

// convertTimelineTransactions converts WebSocket timeline transactions to our Transaction model
func (s *Scraper) convertTimelineTransactions(timelineTransactions []TimelineTransaction, wsClient *WebSocketClient) []models.Transaction {
	transactions := make([]models.Transaction, 0, len(timelineTransactions))

	for _, tt := range timelineTransactions {
		// Convert timestamp - can be int64 (milliseconds) or string
		var timestamp time.Time
		switch v := tt.Timestamp.(type) {
		case float64:
			// Timestamp in milliseconds
			timestamp = time.Unix(0, int64(v)*int64(time.Millisecond))
		case string:
			// Try to parse with different formats
			var err error
			// Format: 2026-01-16T14:54:17.013+0000
			// Note: Go's time package doesn't support variable-length milliseconds in format string
			// So we need to try multiple formats
			formats := []string{
				"2006-01-02T15:04:05.000-0700", // With milliseconds
				"2006-01-02T15:04:05.000Z0700", // With milliseconds, alternative timezone
				time.RFC3339Nano,               // RFC3339 with nanoseconds
				time.RFC3339,                   // Standard RFC3339
				"2006-01-02T15:04:05-0700",     // Without milliseconds
			}

			parsed := false
			for _, format := range formats {
				timestamp, err = time.Parse(format, v)
				if err == nil {
					parsed = true
					break
				}
			}

			if !parsed {
				log.Printf("DEBUG: Failed to parse timestamp with all formats: %v (error: %v)", v, err)
				continue
			}
		default:
			log.Printf("DEBUG: Unknown timestamp type: %T", v)
			continue
		}

		// Extract amount value and currency
		amountValue := 0.0
		amountCurrency := "EUR"
		if tt.Amount != nil {
			if val, ok := tt.Amount["value"].(float64); ok {
				amountValue = val
			}
			if curr, ok := tt.Amount["currency"].(string); ok {
				amountCurrency = curr
			}
		}

		// Extract ISIN from icon path (format: "logos/IE00BM67HM91/v2")
		isin := ""
		if tt.Icon != "" {
			parts := strings.Split(tt.Icon, "/")
			if len(parts) >= 2 {
				// The ISIN is the second part
				potentialISIN := parts[1]
				// Validate it looks like an ISIN (12 characters, starts with 2 letters)
				if len(potentialISIN) == 12 && len(potentialISIN) >= 2 {
					isin = potentialISIN
				}
			}
		}

		// If no ISIN from icon, try action payload
		if isin == "" && tt.Action != nil {
			if payload, ok := tt.Action["payload"].(string); ok {
				// Only use if it looks like an ISIN (12 chars)
				if len(payload) == 12 {
					isin = payload
				}
			}
		}

		// Determine transaction type
		transactionType := s.determineTransactionTypeFromIcon(tt.Icon, tt.Title, tt.Subtitle, amountValue)

		// Convert ISIN to pointer (nil if empty)
		var isinPtr *string
		if isin != "" {
			isinPtr = &isin
		}

		tx := models.Transaction{
			ID:              tt.ID,
			Timestamp:       timestamp.Format(time.RFC3339),
			Title:           tt.Title,
			Subtitle:        tt.Subtitle,
			ISIN:            isinPtr,
			AmountValue:     amountValue,
			AmountCurrency:  amountCurrency,
			Fees:            "0",
			Quantity:        0,
			TransactionType: transactionType,
			Status:          "completed",
			Icon:            tt.Icon,
		}

		// Fetch details for buy/sell transactions to get shares, price, and fees
		if transactionType == "buy" || transactionType == "sell" {
			if err := enrichTransactionWithDetails(&tx, wsClient); err != nil {
				log.Printf("Warning: Failed to fetch details for transaction %s: %v", tx.ID, err)
				// Continue without details rather than failing
			}
		}

		transactions = append(transactions, tx)
	}

	log.Printf("DEBUG: Converted %d timeline transactions to Transaction models", len(transactions))
	return transactions
}

// determineTransactionTypeFromIcon determines the transaction type from icon and title
// determineTransactionTypeFromIcon determines the transaction type from icon, title, subtitle and amount
func (s *Scraper) determineTransactionTypeFromIcon(icon, title, subtitle string, amountValue float64) string {
	iconLower := strings.ToLower(icon)
	titleLower := strings.ToLower(title)
	subtitleLower := strings.ToLower(subtitle)

	// Dividends - check subtitle for "dividende" or "dividend"
	if strings.Contains(subtitleLower, "dividende en espèces") ||
		strings.Contains(subtitleLower, "dividende") ||
		strings.Contains(subtitleLower, "dividend") ||
		strings.Contains(iconLower, "dividend") {
		return "dividend"
	}

	// Interest
	if strings.Contains(titleLower, "intérêts") ||
		strings.Contains(titleLower, "intérêt") ||
		strings.Contains(titleLower, "interest") {
		return "interest"
	}

	// Buy transactions - negative amount means money going out (buying)
	// Check subtitle for execution confirmation or if amount is negative with an ISIN
	if strings.Contains(subtitleLower, "plan d'épargne exécuté") ||
		strings.Contains(subtitleLower, "sparplan ausgeführt") ||
		strings.Contains(subtitleLower, "ordre d'achat") ||
		strings.Contains(subtitleLower, "échec du plan d'épargne") ||
		strings.Contains(subtitleLower, "buy order") ||
		strings.Contains(iconLower, "arrow-right") ||
		strings.Contains(titleLower, "kauf") ||
		strings.Contains(titleLower, "sparplan") {
		return "buy"
	}

	// If amount is negative and title contains an asset name (not "intérêt", "versement", etc.)
	// it's likely a buy transaction
	if amountValue < 0 &&
		!strings.Contains(titleLower, "intérêt") &&
		!strings.Contains(titleLower, "versement") &&
		!strings.Contains(titleLower, "dépôt") &&
		titleLower != "" &&
		// Check if it looks like an asset name (contains letters and possibly numbers)
		len(titleLower) > 3 {
		return "buy"
	}

	// Sell transactions
	if strings.Contains(subtitleLower, "ordre de vente") ||
		strings.Contains(subtitleLower, "sell order") ||
		strings.Contains(iconLower, "arrow-left") ||
		strings.Contains(titleLower, "verkauf") ||
		strings.Contains(titleLower, "vente") {
		return "sell"
	}

	// Deposits - positive amount with specific keywords or "terminé" subtitle
	if strings.Contains(subtitleLower, "terminé") ||
		strings.Contains(titleLower, "einzahlung") ||
		strings.Contains(titleLower, "dépôt") ||
		strings.Contains(titleLower, "versement") ||
		strings.Contains(titleLower, "deposit") {
		// But not if it's a dividend
		if !strings.Contains(subtitleLower, "dividende") {
			return "deposit"
		}
	}

	// If amount is positive and title is a person's name (contains spaces and capital letters)
	// it's likely a deposit
	if amountValue > 0 &&
		strings.Contains(title, " ") &&
		title == strings.Title(strings.ToLower(title)) {
		return "deposit"
	}

	// Withdrawals
	if strings.Contains(titleLower, "auszahlung") ||
		strings.Contains(titleLower, "retrait") ||
		strings.Contains(titleLower, "withdrawal") {
		return "withdrawal"
	}

	// Fees
	if strings.Contains(titleLower, "gebühr") ||
		strings.Contains(titleLower, "frais") ||
		strings.Contains(titleLower, "fee") {
		return "fee"
	}

	return "other"
}

// enrichTransactionWithDetails fetches transaction details and enriches the transaction with shares, price, and fees
func enrichTransactionWithDetails(tx *models.Transaction, wsClient *WebSocketClient) error {
	if wsClient == nil {
		return fmt.Errorf("WebSocket client not initialized")
	}

	// Fetch transaction detail
	detail, err := wsClient.FetchTransactionDetail(tx.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch transaction detail: %w", err)
	}

	// Extract shares and price
	shares, sharePrice, err := ExtractSharesAndPriceFromDetail(detail)
	if err != nil {
		// Not all transactions have shares/price (e.g., some corporate actions)
		// This is not necessarily an error
		fmt.Printf("Info: Could not extract shares/price for transaction %s: %v\n", tx.ID, err)
	} else {
		// Store shares and share_price as strings for now (model uses string fields)
		tx.Shares = fmt.Sprintf("%.2f", shares)
		tx.SharePrice = fmt.Sprintf("%.2f", sharePrice)
		tx.Quantity = shares // Quantity is the number of shares
	}

	// Extract fees
	feesStr := ExtractFeesFromDetail(detail)
	if feesStr != "0" {
		tx.Fees = feesStr
	}

	return nil
}
