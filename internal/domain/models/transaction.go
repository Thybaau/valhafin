package models

import (
	"errors"
	"time"
)

type Transaction struct {
	ID                string  `json:"id" csv:"id" db:"id"`
	Timestamp         string  `json:"timestamp" csv:"timestamp" db:"timestamp"`
	Title             string  `json:"title" csv:"title" db:"title"`
	Icon              string  `json:"icon" csv:"icon" db:"icon"`
	Avatar            string  `json:"avatar" csv:"avatar.asset" db:"avatar"`
	Subtitle          string  `json:"subtitle" csv:"subtitle" db:"subtitle"`
	AmountCurrency    string  `json:"amount_currency" csv:"amount.currency" db:"amount_currency"`
	AmountValue       float64 `json:"amount_value" csv:"amount.value" db:"amount_value"`
	AmountFraction    int     `json:"amount_fraction" csv:"amount.fractionDigits" db:"amount_fraction"`
	Status            string  `json:"status" csv:"status" db:"status"`
	ActionType        string  `json:"action_type" csv:"action.type" db:"action_type"`
	ActionPayload     string  `json:"action_payload" csv:"action.payload" db:"action_payload"`
	CashAccountNumber string  `json:"cash_account_number" csv:"cashAccountNumber" db:"cash_account_number"`
	Hidden            bool    `json:"hidden" csv:"hidden" db:"hidden"`
	Deleted           bool    `json:"deleted" csv:"deleted" db:"deleted"`

	// Details (when extract_details is true)
	Actions          string `json:"actions,omitempty" csv:"Actions" db:"actions"`
	DividendPerShare string `json:"dividend_per_share,omitempty" csv:"Dividende par action" db:"dividend_per_share"`
	Taxes            string `json:"taxes,omitempty" csv:"Taxes" db:"taxes"`
	Total            string `json:"total,omitempty" csv:"Total" db:"total"`
	Shares           string `json:"shares,omitempty" csv:"Titres" db:"shares"`
	SharePrice       string `json:"share_price,omitempty" csv:"Cours du titre" db:"share_price"`
	Fees             string `json:"fees,omitempty" csv:"Frais" db:"fees"`
	Amount           string `json:"amount,omitempty" csv:"Montant" db:"amount"`

	// New fields for database integration
	AccountID       string  `json:"account_id" db:"account_id"`
	ISIN            *string `json:"isin,omitempty" db:"isin"`
	Quantity        float64 `json:"quantity,omitempty" db:"quantity"`
	TransactionType string  `json:"transaction_type,omitempty" db:"transaction_type"` // "buy", "sell", "dividend", "fee"
	Metadata        *string `json:"metadata,omitempty" db:"metadata"`                 // JSON string for additional platform-specific data
}

// Validate validates the Transaction model
func (t *Transaction) Validate() error {
	if t.ID == "" {
		return errors.New("transaction ID is required")
	}

	if t.AccountID == "" {
		return errors.New("account ID is required")
	}

	if t.Timestamp == "" {
		return errors.New("timestamp is required")
	}

	// Validate timestamp format
	_, err := time.Parse(time.RFC3339, t.Timestamp)
	if err != nil {
		return errors.New("timestamp must be in RFC3339 format")
	}

	if t.AmountCurrency == "" {
		return errors.New("amount currency is required")
	}

	return nil
}

type ProfileCash struct {
	Currency       string  `json:"currency" csv:"currency"`
	Value          float64 `json:"value" csv:"value"`
	FractionDigits int     `json:"fractionDigits" csv:"fractionDigits"`
}
