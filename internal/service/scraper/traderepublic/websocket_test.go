package traderepublic

import (
	"encoding/json"
	"testing"
)

func TestExtractSharesAndPriceFromDetail(t *testing.T) {
	// Test data from pytr buy.json
	jsonData := `{
		"id": "12345678",
		"sections": [
			{
				"title": "Du hast 3.002,80 €  investiert",
				"type": "header"
			},
			{
				"title": "Übersicht",
				"type": "table",
				"data": [
					{
						"title": "Status",
						"detail": {
							"text": "Ausgeführt"
						}
					}
				]
			},
			{
				"title": "Transaktion",
				"type": "table",
				"data": [
					{
						"title": "Anteile",
						"detail": {
							"text": "60"
						}
					},
					{
						"title": "Aktienkurs",
						"detail": {
							"text": "50,03 €"
						}
					},
					{
						"title": "Gebühr",
						"detail": {
							"text": "1,00 €"
						}
					},
					{
						"title": "Gesamt",
						"detail": {
							"text": "3.002,80 €"
						}
					}
				]
			}
		]
	}`

	var detail TimelineDetail
	if err := json.Unmarshal([]byte(jsonData), &detail); err != nil {
		t.Fatalf("Failed to unmarshal test data: %v", err)
	}

	shares, sharePrice, err := ExtractSharesAndPriceFromDetail(&detail)
	if err != nil {
		t.Fatalf("ExtractSharesAndPriceFromDetail failed: %v", err)
	}

	expectedShares := 60.0
	expectedPrice := 50.03

	if shares != expectedShares {
		t.Errorf("Expected shares %f, got %f", expectedShares, shares)
	}

	if sharePrice != expectedPrice {
		t.Errorf("Expected share price %f, got %f", expectedPrice, sharePrice)
	}

	t.Logf("✓ Successfully extracted shares: %f, price: %f", shares, sharePrice)
}

func TestExtractFeesFromDetail(t *testing.T) {
	jsonData := `{
		"id": "12345678",
		"sections": [
			{
				"title": "Transaktion",
				"type": "table",
				"data": [
					{
						"title": "Anteile",
						"detail": {
							"text": "60"
						}
					},
					{
						"title": "Gebühr",
						"detail": {
							"text": "1,00 €"
						}
					}
				]
			}
		]
	}`

	var detail TimelineDetail
	if err := json.Unmarshal([]byte(jsonData), &detail); err != nil {
		t.Fatalf("Failed to unmarshal test data: %v", err)
	}

	fees := ExtractFeesFromDetail(&detail)
	expectedFees := "1.00"

	if fees != expectedFees {
		t.Errorf("Expected fees %s, got %s", expectedFees, fees)
	}

	t.Logf("✓ Successfully extracted fees: %s", fees)
}
