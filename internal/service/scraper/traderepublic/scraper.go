package traderepublic

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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
	transactions := s.convertTimelineTransactions(timelineTransactions)

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
func (s *Scraper) convertTimelineTransactions(timelineTransactions []TimelineTransaction) []models.Transaction {
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
		transactionType := s.determineTransactionTypeFromIcon(tt.Icon, tt.Title)

		tx := models.Transaction{
			ID:              tt.ID,
			Timestamp:       timestamp.Format(time.RFC3339),
			Title:           tt.Title,
			Subtitle:        tt.Subtitle,
			ISIN:            isin,
			AmountValue:     amountValue,
			AmountCurrency:  amountCurrency,
			Fees:            "0",
			Quantity:        0,
			TransactionType: transactionType,
			Status:          "completed",
			Icon:            tt.Icon,
		}

		transactions = append(transactions, tx)
	}

	log.Printf("DEBUG: Converted %d timeline transactions to Transaction models", len(transactions))
	return transactions
}

// determineTransactionTypeFromIcon determines the transaction type from icon and title
func (s *Scraper) determineTransactionTypeFromIcon(icon, title string) string {
	iconLower := strings.ToLower(icon)
	titleLower := strings.ToLower(title)

	if strings.Contains(iconLower, "dividend") || strings.Contains(titleLower, "dividende") {
		return "dividend"
	}
	if strings.Contains(iconLower, "arrow-right") || strings.Contains(titleLower, "kauf") || strings.Contains(titleLower, "sparplan") {
		return "buy"
	}
	if strings.Contains(iconLower, "arrow-left") || strings.Contains(titleLower, "verkauf") {
		return "sell"
	}
	if strings.Contains(titleLower, "einzahlung") {
		return "deposit"
	}
	if strings.Contains(titleLower, "auszahlung") {
		return "withdrawal"
	}
	if strings.Contains(titleLower, "gebühr") {
		return "fee"
	}

	return "other"
}

// TimelineEvent represents a Trade Republic timeline event
type TimelineEvent struct {
	Type string `json:"type"`
	Data struct {
		ID               string  `json:"id"`
		Timestamp        int64   `json:"timestamp"`
		Icon             string  `json:"icon"`
		Title            string  `json:"title"`
		Body             string  `json:"body"`
		CashChangeAmount float64 `json:"cashChangeAmount"`
		Action           struct {
			Type    string `json:"type"`
			Payload string `json:"payload"`
		} `json:"action"`
		Month string `json:"month"`
	} `json:"data"`
}

// fetchTimelineEvents fetches timeline events from Trade Republic API
func (s *Scraper) fetchTimelineEvents(sessionToken string) ([]TimelineEvent, error) {
	// Trade Republic timeline endpoint
	timelineURL := baseURL + "/api/v1/timeline"

	req, err := http.NewRequest("GET", timelineURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cookie", "tr_session="+sessionToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch timeline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch timeline: status %d, body: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read timeline response: %w", err)
	}

	fmt.Printf("DEBUG: Downloaded timeline data, size: %d bytes\n", len(data))

	// Parse timeline events
	var events []TimelineEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("failed to parse timeline events: %w", err)
	}

	fmt.Printf("DEBUG: Parsed %d timeline events\n", len(events))
	return events, nil
}

// parseTimelineEvents converts timeline events to Transaction models
func (s *Scraper) parseTimelineEvents(events []TimelineEvent) []models.Transaction {
	transactions := make([]models.Transaction, 0, len(events))

	for _, event := range events {
		if event.Type != "timelineEvent" {
			continue
		}

		// Convert timestamp from milliseconds to time.Time
		timestamp := time.Unix(0, event.Data.Timestamp*int64(time.Millisecond))

		// Determine transaction type from icon or body
		transactionType := s.determineTransactionTypeFromEvent(event)

		// Extract ISIN from action payload if available
		isin := ""
		if event.Data.Action.Type == "timelineDetail" {
			isin = event.Data.Action.Payload
		}

		tx := models.Transaction{
			ID:              event.Data.ID,
			Timestamp:       timestamp.Format(time.RFC3339),
			Title:           event.Data.Title,
			Subtitle:        event.Data.Body,
			ISIN:            isin,
			AmountValue:     event.Data.CashChangeAmount,
			AmountCurrency:  "EUR", // Trade Republic uses EUR
			Fees:            "0",   // Fees are included in the cash change amount
			Quantity:        0,     // Not provided in timeline events
			TransactionType: transactionType,
			Status:          "completed",
			Icon:            event.Data.Icon,
		}

		transactions = append(transactions, tx)
	}

	fmt.Printf("DEBUG: Converted %d timeline events to transactions\n", len(transactions))
	return transactions
}

// determineTransactionTypeFromEvent determines the transaction type from a timeline event
func (s *Scraper) determineTransactionTypeFromEvent(event TimelineEvent) string {
	body := strings.ToLower(event.Data.Body)
	icon := strings.ToLower(event.Data.Icon)

	if strings.Contains(icon, "dividend") || strings.Contains(body, "dividende") {
		return "dividend"
	}
	if strings.Contains(icon, "arrow-right") || strings.Contains(body, "kauf") || strings.Contains(body, "sparplan") {
		return "buy"
	}
	if strings.Contains(icon, "arrow-left") || strings.Contains(body, "verkauf") {
		return "sell"
	}
	if strings.Contains(body, "einzahlung") {
		return "deposit"
	}
	if strings.Contains(body, "auszahlung") {
		return "withdrawal"
	}
	if strings.Contains(body, "gebühr") {
		return "fee"
	}

	return "other"
}

// downloadCSVExport downloads the CSV export from Trade Republic (DEPRECATED - kept for reference)
func (s *Scraper) downloadCSVExport(sessionToken string) ([]byte, error) {
	// Trade Republic CSV export endpoint
	exportURL := baseURL + "/api/v1/timeline/export"

	req, err := http.NewRequest("GET", exportURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cookie", "tr_session="+sessionToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download CSV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download CSV: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV response: %w", err)
	}

	fmt.Printf("DEBUG: Downloaded CSV data, size: %d bytes\n", len(data))
	return data, nil
}

// parseCSVTransactions parses CSV data into Transaction models
func (s *Scraper) parseCSVTransactions(csvData []byte) ([]models.Transaction, error) {
	reader := csv.NewReader(bytes.NewReader(csvData))
	reader.Comma = ','
	reader.LazyQuotes = true

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	fmt.Printf("DEBUG: CSV has %d records (including header)\n", len(records))

	if len(records) == 0 {
		return []models.Transaction{}, nil
	}

	// Print header for debugging
	if len(records) > 0 {
		fmt.Printf("DEBUG: CSV header: %v\n", records[0])
	}

	// Skip header row
	if len(records) > 0 {
		records = records[1:]
	}

	transactions := make([]models.Transaction, 0, len(records))

	for i, record := range records {
		if len(record) < 8 {
			fmt.Printf("DEBUG: Skipping row %d - insufficient columns (%d)\n", i+1, len(record))
			continue // Skip invalid rows
		}

		// Parse timestamp
		timestamp, err := time.Parse("2006-01-02 15:04:05", record[0])
		if err != nil {
			// Try alternative format
			timestamp, err = time.Parse("02.01.2006", record[0])
			if err != nil {
				fmt.Printf("DEBUG: Skipping row %d - invalid timestamp: %s\n", i+1, record[0])
				continue
			}
		}

		// Parse amount
		amountStr := strings.ReplaceAll(record[4], ",", ".")
		amountStr = strings.TrimSpace(amountStr)
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			fmt.Printf("DEBUG: Row %d - failed to parse amount: %s\n", i+1, record[4])
			amount = 0
		}

		// Parse fees if present
		feesStr := "0"
		if len(record) > 7 && record[7] != "" {
			feesStr = strings.ReplaceAll(record[7], ",", ".")
			feesStr = strings.TrimSpace(feesStr)
		}

		// Parse quantity if present
		quantity := 0.0
		if len(record) > 6 && record[6] != "" {
			quantityStr := strings.ReplaceAll(record[6], ",", ".")
			quantityStr = strings.TrimSpace(quantityStr)
			quantity, _ = strconv.ParseFloat(quantityStr, 64)
		}

		tx := models.Transaction{
			ID:              fmt.Sprintf("tr_%d", timestamp.Unix()),
			Timestamp:       timestamp.Format(time.RFC3339),
			Title:           record[1],
			Subtitle:        record[2],
			ISIN:            record[3],
			AmountValue:     amount,
			AmountCurrency:  record[5],
			Fees:            feesStr,
			Quantity:        quantity,
			TransactionType: s.determineTransactionType(record[1]),
			Status:          "completed",
		}

		transactions = append(transactions, tx)
	}

	fmt.Printf("DEBUG: Parsed %d transactions from CSV\n", len(transactions))
	return transactions, nil
}

// determineTransactionType determines the transaction type from the title
func (s *Scraper) determineTransactionType(title string) string {
	title = strings.ToLower(title)

	if strings.Contains(title, "kauf") || strings.Contains(title, "buy") {
		return "buy"
	}
	if strings.Contains(title, "verkauf") || strings.Contains(title, "sell") {
		return "sell"
	}
	if strings.Contains(title, "dividende") || strings.Contains(title, "dividend") {
		return "dividend"
	}
	if strings.Contains(title, "gebühr") || strings.Contains(title, "fee") {
		return "fee"
	}
	if strings.Contains(title, "einzahlung") || strings.Contains(title, "deposit") {
		return "deposit"
	}
	if strings.Contains(title, "auszahlung") || strings.Contains(title, "withdrawal") {
		return "withdrawal"
	}

	return "other"
}
