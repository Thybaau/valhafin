package traderepublic

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"valhafin/models"
)

type localeConfig struct {
	Locale          string `json:"locale"`
	PlatformID      string `json:"platformId"`
	PlatformVersion string `json:"platformVersion"`
	ClientID        string `json:"clientId"`
	ClientVersion   string `json:"clientVersion"`
}

func connectWebSocket() (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, err
	}

	// Send connection message
	locale := localeConfig{
		Locale:          "fr",
		PlatformID:      "webtrading",
		PlatformVersion: "safari - 18.3.0",
		ClientID:        "app.traderepublic.com",
		ClientVersion:   "3.151.3",
	}

	localeJSON, _ := json.Marshal(locale)
	connectMsg := fmt.Sprintf("connect 31 %s", string(localeJSON))

	if err := conn.WriteMessage(websocket.TextMessage, []byte(connectMsg)); err != nil {
		return nil, err
	}

	// Read connection response
	_, _, err = conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	fmt.Println("✅ WebSocket connection successful!\n⏳ Please wait...")
	return conn, nil
}

func (s *Scraper) FetchTransactions(token string) ([]models.Transaction, error) {
	conn, err := connectWebSocket()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var allTransactions []models.Transaction
	messageID := 0
	var afterCursor *string

	for {
		payload := map[string]interface{}{
			"type":  "timelineTransactions",
			"token": token,
		}

		if afterCursor != nil {
			payload["after"] = *afterCursor
		}

		messageID++
		payloadJSON, _ := json.Marshal(payload)
		subMsg := fmt.Sprintf("sub %d %s", messageID, string(payloadJSON))

		if err := conn.WriteMessage(websocket.TextMessage, []byte(subMsg)); err != nil {
			return nil, err
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return nil, err
		}

		unsubMsg := fmt.Sprintf("unsub %d", messageID)
		conn.WriteMessage(websocket.TextMessage, []byte(unsubMsg))
		conn.ReadMessage()

		// Parse response
		response := string(message)
		startIdx := strings.Index(response, "{")
		endIdx := strings.LastIndex(response, "}")

		if startIdx == -1 || endIdx == -1 {
			break
		}

		jsonData := response[startIdx : endIdx+1]

		var data struct {
			Items   []map[string]interface{} `json:"items"`
			Cursors struct {
				After string `json:"after"`
			} `json:"cursors"`
		}

		if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
			return nil, err
		}

		if len(data.Items) == 0 {
			break
		}

		// Process transactions
		for _, item := range data.Items {
			transaction := s.parseTransaction(item)

			// Fetch details if enabled
			if s.config.General.ExtractDetails {
				details, err := s.fetchTransactionDetails(conn, transaction.ID, token, &messageID)
				if err == nil {
					s.mergeDetails(&transaction, details)
				}
			}

			allTransactions = append(allTransactions, transaction)
		}

		if data.Cursors.After == "" {
			break
		}
		afterCursor = &data.Cursors.After
	}

	return allTransactions, nil
}

func (s *Scraper) fetchTransactionDetails(conn *websocket.Conn, transactionID, token string, messageID *int) (map[string]string, error) {
	payload := map[string]interface{}{
		"type":  "timelineDetailV2",
		"id":    transactionID,
		"token": token,
	}

	*messageID++
	payloadJSON, _ := json.Marshal(payload)
	subMsg := fmt.Sprintf("sub %d %s", *messageID, string(payloadJSON))

	if err := conn.WriteMessage(websocket.TextMessage, []byte(subMsg)); err != nil {
		return nil, err
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	unsubMsg := fmt.Sprintf("unsub %d", *messageID)
	conn.WriteMessage(websocket.TextMessage, []byte(unsubMsg))
	conn.ReadMessage()

	// Parse response
	response := string(message)
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")

	if startIdx == -1 || endIdx == -1 {
		return nil, fmt.Errorf("invalid response")
	}

	jsonData := response[startIdx : endIdx+1]

	var data struct {
		Sections []struct {
			Title string `json:"title"`
			Data  []struct {
				Title  string `json:"title"`
				Detail struct {
					Text string `json:"text"`
				} `json:"detail"`
			} `json:"data"`
		} `json:"sections"`
	}

	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, err
	}

	details := make(map[string]string)
	for _, section := range data.Sections {
		if section.Title == "Transaction" {
			for _, item := range section.Data {
				if item.Title != "" && item.Detail.Text != "" {
					details[item.Title] = item.Detail.Text
				}
			}
		}
	}

	return details, nil
}

func (s *Scraper) FetchProfileCash(token string) ([]models.ProfileCash, error) {
	conn, err := connectWebSocket()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	payload := map[string]interface{}{
		"type":  "availableCash",
		"token": token,
	}

	payloadJSON, _ := json.Marshal(payload)
	subMsg := fmt.Sprintf("sub 1 %s", string(payloadJSON))

	if err := conn.WriteMessage(websocket.TextMessage, []byte(subMsg)); err != nil {
		return nil, err
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	// Parse response
	response := string(message)
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")

	if startIdx == -1 || endIdx == -1 {
		return nil, fmt.Errorf("invalid response")
	}

	jsonData := response[startIdx : endIdx+1]

	var cashData []models.ProfileCash
	if err := json.Unmarshal([]byte(jsonData), &cashData); err != nil {
		return nil, err
	}

	return cashData, nil
}

func (s *Scraper) parseTransaction(item map[string]interface{}) models.Transaction {
	transaction := models.Transaction{
		ID:        getString(item, "id"),
		Timestamp: formatTimestamp(getInt64(item, "timestamp")),
		Title:     getString(item, "title"),
		Icon:      getString(item, "icon"),
		Subtitle:  getString(item, "subtitle"),
		Status:    getString(item, "status"),
		Hidden:    getBool(item, "hidden"),
		Deleted:   getBool(item, "deleted"),
	}

	if avatar, ok := item["avatar"].(map[string]interface{}); ok {
		transaction.Avatar = getString(avatar, "asset")
	}

	if amount, ok := item["amount"].(map[string]interface{}); ok {
		transaction.AmountCurrency = getString(amount, "currency")
		transaction.AmountValue = getFloat64(amount, "value")
		transaction.AmountFraction = getInt(amount, "fractionDigits")
	}

	if action, ok := item["action"].(map[string]interface{}); ok {
		transaction.ActionType = getString(action, "type")
		transaction.ActionPayload = getString(action, "payload")
	}

	transaction.CashAccountNumber = getString(item, "cashAccountNumber")

	return transaction
}

func (s *Scraper) mergeDetails(transaction *models.Transaction, details map[string]string) {
	transaction.Actions = details["Actions"]
	transaction.DividendPerShare = details["Dividende par action"]
	transaction.Taxes = details["Taxes"]
	transaction.Total = details["Total"]
	transaction.Shares = details["Titres"]
	transaction.SharePrice = details["Cours du titre"]
	transaction.Fees = details["Frais"]
	transaction.Amount = details["Montant"]
}

func formatTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	t := time.Unix(timestamp/1000, 0)
	return t.Format("02/01/2006")
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
