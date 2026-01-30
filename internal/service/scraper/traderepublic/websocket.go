package traderepublic

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketClient handles WebSocket communication with Trade Republic
type WebSocketClient struct {
	conn         *websocket.Conn
	sessionToken string
	messageID    int
}

// NewWebSocketClient creates a new WebSocket client and connects
func NewWebSocketClient(sessionToken string) (*WebSocketClient, error) {
	// Connect to Trade Republic WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	client := &WebSocketClient{
		conn:         conn,
		sessionToken: sessionToken,
		messageID:    0,
	}

	// Send connect message
	localeConfig := map[string]interface{}{
		"locale":          "fr",
		"platformId":      "webtrading",
		"platformVersion": "safari - 18.3.0",
		"clientId":        "app.traderepublic.com",
		"clientVersion":   "3.151.3",
	}

	configJSON, _ := json.Marshal(localeConfig)
	connectMsg := fmt.Sprintf("connect 31 %s", string(configJSON))

	if err := conn.WriteMessage(websocket.TextMessage, []byte(connectMsg)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send connect message: %w", err)
	}

	// Read connect response
	_, _, err = conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read connect response: %w", err)
	}

	log.Printf("DEBUG: WebSocket connected successfully")

	return client, nil
}

// Close closes the WebSocket connection
func (c *WebSocketClient) Close() error {
	return c.conn.Close()
}

// TimelineTransaction represents a transaction from the timeline
type TimelineTransaction struct {
	ID        string                 `json:"id"`
	Timestamp interface{}            `json:"timestamp"` // Can be int64 or string
	Title     string                 `json:"title"`
	Subtitle  string                 `json:"subtitle"`
	Amount    map[string]interface{} `json:"amount"`
	Icon      string                 `json:"icon"`
	Action    map[string]interface{} `json:"action"`
}

// TimelineDetail represents detailed information about a transaction
type TimelineDetail struct {
	Sections []struct {
		Type string `json:"type"`
		Data struct {
			Items []struct {
				Title  string `json:"title"`
				Detail struct {
					Text string `json:"text"`
				} `json:"detail"`
			} `json:"items"`
		} `json:"data"`
	} `json:"sections"`
}

// TimelineResponse represents the response from timelineTransactions
type TimelineResponse struct {
	Items   []TimelineTransaction  `json:"items"`
	Cursors map[string]interface{} `json:"cursors"`
}

// FetchTimeline fetches all timeline transactions via WebSocket
func (c *WebSocketClient) FetchTimeline() ([]TimelineTransaction, error) {
	allTransactions := []TimelineTransaction{}
	var afterCursor interface{}

	for {
		c.messageID++

		// Build payload
		payload := map[string]interface{}{
			"type":  "timelineTransactions",
			"token": c.sessionToken,
		}
		if afterCursor != nil {
			payload["after"] = afterCursor
		}

		payloadJSON, _ := json.Marshal(payload)
		subMsg := fmt.Sprintf("sub %d %s", c.messageID, string(payloadJSON))

		// Send subscription
		if err := c.conn.WriteMessage(websocket.TextMessage, []byte(subMsg)); err != nil {
			return nil, fmt.Errorf("failed to send subscription: %w", err)
		}

		// Read response
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		// Send unsubscribe
		unsubMsg := fmt.Sprintf("unsub %d", c.messageID)
		if err := c.conn.WriteMessage(websocket.TextMessage, []byte(unsubMsg)); err != nil {
			return nil, fmt.Errorf("failed to send unsubscribe: %w", err)
		}

		// Read unsubscribe response
		_, _, err = c.conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("failed to read unsubscribe response: %w", err)
		}

		// Parse response - extract JSON from message
		messageStr := string(message)
		startIndex := strings.Index(messageStr, "{")
		endIndex := strings.LastIndex(messageStr, "}")

		if startIndex == -1 || endIndex == -1 {
			log.Printf("DEBUG: No JSON found in message: %s", messageStr)
			break
		}

		jsonStr := messageStr[startIndex : endIndex+1]
		log.Printf("DEBUG: Received JSON: %s", jsonStr[:min(200, len(jsonStr))]) // Log first 200 chars

		var response TimelineResponse
		if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
			log.Printf("DEBUG: Failed to parse response: %v", err)
			log.Printf("DEBUG: JSON was: %s", jsonStr)
			break
		}

		if len(response.Items) == 0 {
			break
		}

		allTransactions = append(allTransactions, response.Items...)

		// Check for next page
		if response.Cursors != nil {
			if after, ok := response.Cursors["after"]; ok && after != nil {
				afterCursor = after
			} else {
				break
			}
		} else {
			break
		}
	}

	log.Printf("DEBUG: Fetched %d transactions from WebSocket", len(allTransactions))
	return allTransactions, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// FetchTransactionDetail fetches detailed information for a specific transaction
func (c *WebSocketClient) FetchTransactionDetail(transactionID string) (*TimelineDetail, error) {
	c.messageID++

	// Build payload for timelineDetail
	payload := map[string]interface{}{
		"type":  "timelineDetail",
		"id":    transactionID,
		"token": c.sessionToken,
	}

	payloadJSON, _ := json.Marshal(payload)
	subMsg := fmt.Sprintf("sub %d %s", c.messageID, string(payloadJSON))

	// Send subscription
	if err := c.conn.WriteMessage(websocket.TextMessage, []byte(subMsg)); err != nil {
		return nil, fmt.Errorf("failed to send subscription: %w", err)
	}

	// Read response
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Send unsubscribe
	unsubMsg := fmt.Sprintf("unsub %d", c.messageID)
	if err := c.conn.WriteMessage(websocket.TextMessage, []byte(unsubMsg)); err != nil {
		return nil, fmt.Errorf("failed to send unsubscribe: %w", err)
	}

	// Read unsubscribe response
	_, _, err = c.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read unsubscribe response: %w", err)
	}

	// Parse response - extract JSON from message
	messageStr := string(message)
	startIndex := strings.Index(messageStr, "{")
	endIndex := strings.LastIndex(messageStr, "}")

	if startIndex == -1 || endIndex == -1 {
		return nil, fmt.Errorf("no JSON found in message")
	}

	jsonStr := messageStr[startIndex : endIndex+1]

	var detail TimelineDetail
	if err := json.Unmarshal([]byte(jsonStr), &detail); err != nil {
		return nil, fmt.Errorf("failed to parse detail response: %w", err)
	}

	return &detail, nil
}

// ExtractFeesFromDetail extracts fees from transaction detail
func ExtractFeesFromDetail(detail *TimelineDetail) string {
	if detail == nil {
		return "0"
	}

	// Look for "Frais" or "Gebühren" in the sections
	for _, section := range detail.Sections {
		if section.Type == "table" {
			for _, item := range section.Data.Items {
				titleLower := strings.ToLower(item.Title)
				if strings.Contains(titleLower, "frais") ||
					strings.Contains(titleLower, "gebühren") ||
					strings.Contains(titleLower, "fee") {
					// Extract numeric value from detail text
					// Format is usually like "1,00 €" or "1.00 EUR"
					text := item.Detail.Text
					// Remove currency symbols and convert comma to dot
					text = strings.ReplaceAll(text, "€", "")
					text = strings.ReplaceAll(text, "EUR", "")
					text = strings.ReplaceAll(text, ",", ".")
					text = strings.TrimSpace(text)
					return text
				}
			}
		}
	}

	return "0"
}
