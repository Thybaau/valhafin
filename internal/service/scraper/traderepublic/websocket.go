package traderepublic

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
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

// TimelineDetail represents detailed information about a transaction (V2)
type TimelineDetail struct {
	ID       string `json:"id"`
	Sections []struct {
		Title  string      `json:"title"`
		Type   string      `json:"type"`
		Data   interface{} `json:"data"` // Can be array or object
		Action interface{} `json:"action"`
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

	// Build payload for timelineDetailV2
	payload := map[string]interface{}{
		"type":  "timelineDetailV2",
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

	// Look for "Frais" or "Gebühren" or "Fee" in the sections
	for _, section := range detail.Sections {
		if section.Type == "table" {
			// Parse data as array of items
			if dataArray, ok := section.Data.([]interface{}); ok {
				for _, item := range dataArray {
					if itemMap, ok := item.(map[string]interface{}); ok {
						title, _ := itemMap["title"].(string)
						titleLower := strings.ToLower(title)
						// Check for fees in multiple languages
						if strings.Contains(titleLower, "gebühr") ||
							strings.Contains(titleLower, "fee") ||
							strings.Contains(titleLower, "frais") {
							// Extract numeric value from detail text
							if detail, ok := itemMap["detail"].(map[string]interface{}); ok {
								if text, ok := detail["text"].(string); ok {
									// Remove currency symbols and convert comma to dot
									text = strings.ReplaceAll(text, "€", "")
									text = strings.ReplaceAll(text, "EUR", "")
									text = strings.ReplaceAll(text, " ", "")
									text = strings.ReplaceAll(text, ",", ".")
									text = strings.TrimSpace(text)
									log.Printf("DEBUG ExtractFees: Found fees: %s", text)
									return text
								}
							}
						}
					}
				}
			}
		}
	}

	return "0"
}

// ExtractSharesAndPriceFromDetail extracts shares quantity and share price from transaction detail V2
func ExtractSharesAndPriceFromDetail(detail *TimelineDetail) (shares float64, sharePrice float64, err error) {
	if detail == nil {
		return 0, 0, fmt.Errorf("detail is nil")
	}

	log.Printf("DEBUG ExtractSharesAndPrice: Processing detail with %d sections", len(detail.Sections))

	// Look for "Transaktion" or "Synthèse" section
	for i, section := range detail.Sections {
		log.Printf("DEBUG ExtractSharesAndPrice: Section %d - Type: %s, Title: %s", i, section.Type, section.Title)

		if section.Type == "table" {
			// Parse data as array of items
			if dataArray, ok := section.Data.([]interface{}); ok {
				log.Printf("DEBUG ExtractSharesAndPrice: Section %d has %d items", i, len(dataArray))

				var sharesStr, priceStr string

				for j, item := range dataArray {
					if itemMap, ok := item.(map[string]interface{}); ok {
						title, _ := itemMap["title"].(string)
						log.Printf("DEBUG ExtractSharesAndPrice: Section %d, Item %d - Title: %s", i, j, title)

						// NEW FORMAT: Check if this item is "Transaction" with embedded sections
						if title == "Transaction" {
							if detailMap, ok := itemMap["detail"].(map[string]interface{}); ok {
								if action, ok := detailMap["action"].(map[string]interface{}); ok {
									if payload, ok := action["payload"].(map[string]interface{}); ok {
										if embeddedSections, ok := payload["sections"].([]interface{}); ok {
											log.Printf("DEBUG ExtractSharesAndPrice: Found embedded sections in Transaction item (new format)")
											// Parse embedded sections
											sharesStr, priceStr = extractFromEmbeddedSections(embeddedSections)
											if sharesStr != "" && priceStr != "" {
												goto parseValues
											}
										}
									}
								}
							}
						}

						// OLD FORMAT: Look for "Anteile" or "Aktien" or "Titres" or "Actions"
						if title == "Anteile" || title == "Aktien" || title == "Titres" || title == "Actions" {
							if detail, ok := itemMap["detail"].(map[string]interface{}); ok {
								if text, ok := detail["text"].(string); ok {
									sharesStr = text
									log.Printf("DEBUG ExtractSharesAndPrice: Found shares (old format): %s", sharesStr)
								}
							}
						}

						// OLD FORMAT: Look for "Aktienkurs" or "Kurs" or "Cours du titre" or "Prix du titre"
						if title == "Aktienkurs" || title == "Kurs" || title == "Cours du titre" || title == "Prix du titre" {
							if detail, ok := itemMap["detail"].(map[string]interface{}); ok {
								if text, ok := detail["text"].(string); ok {
									priceStr = text
									log.Printf("DEBUG ExtractSharesAndPrice: Found price (old format): %s", priceStr)
								}
							}
						}
					}
				}

			parseValues:
				// Parse shares
				if sharesStr != "" {
					sharesStr = strings.ReplaceAll(sharesStr, ",", ".")
					sharesStr = strings.ReplaceAll(sharesStr, " ", "")
					sharesStr = strings.TrimSpace(sharesStr)
					if s, err := strconv.ParseFloat(sharesStr, 64); err == nil {
						shares = s
					} else {
						log.Printf("DEBUG ExtractSharesAndPrice: Failed to parse shares '%s': %v", sharesStr, err)
					}
				}

				// Parse share price
				if priceStr != "" {
					priceStr = strings.ReplaceAll(priceStr, "€", "")
					priceStr = strings.ReplaceAll(priceStr, "EUR", "")
					priceStr = strings.ReplaceAll(priceStr, " ", "")
					priceStr = strings.ReplaceAll(priceStr, ",", ".")
					priceStr = strings.TrimSpace(priceStr)
					if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
						sharePrice = p
					} else {
						log.Printf("DEBUG ExtractSharesAndPrice: Failed to parse price '%s': %v", priceStr, err)
					}
				}

				if shares > 0 && sharePrice > 0 {
					log.Printf("DEBUG ExtractSharesAndPrice: Successfully extracted shares=%.2f, price=%.2f", shares, sharePrice)
					return shares, sharePrice, nil
				}
			}
		}
	}

	log.Printf("DEBUG ExtractSharesAndPrice: Failed to extract shares and price")
	return 0, 0, fmt.Errorf("could not extract shares and price from detail")
}

// extractFromEmbeddedSections extracts shares and price from embedded sections (new format)
func extractFromEmbeddedSections(sections []interface{}) (sharesStr, priceStr string) {
	for _, sec := range sections {
		if secMap, ok := sec.(map[string]interface{}); ok {
			secType, _ := secMap["type"].(string)
			if secType == "table" {
				if data, ok := secMap["data"].([]interface{}); ok {
					for _, item := range data {
						if itemMap, ok := item.(map[string]interface{}); ok {
							title, _ := itemMap["title"].(string)
							log.Printf("DEBUG extractFromEmbeddedSections: Item title: %s", title)

							// Look for "Actions" (French) or "Anteile" (German) or "Aktien"
							if title == "Actions" || title == "Anteile" || title == "Aktien" {
								if detail, ok := itemMap["detail"].(map[string]interface{}); ok {
									if text, ok := detail["text"].(string); ok {
										sharesStr = text
										log.Printf("DEBUG extractFromEmbeddedSections: Found shares: %s", sharesStr)
									}
								}
							}

							// Look for "Prix du titre" (French) or "Aktienkurs" (German) or "Kurs"
							if title == "Prix du titre" || title == "Aktienkurs" || title == "Kurs" {
								if detail, ok := itemMap["detail"].(map[string]interface{}); ok {
									if text, ok := detail["text"].(string); ok {
										priceStr = text
										log.Printf("DEBUG extractFromEmbeddedSections: Found price: %s", priceStr)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return sharesStr, priceStr
}
