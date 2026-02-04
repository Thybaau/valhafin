package models

import (
	"testing"
	"time"
)

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

func TestAccountValidation(t *testing.T) {
	tests := []struct {
		name    string
		account Account
		wantErr bool
	}{
		{
			name: "valid account",
			account: Account{
				Name:        "My Trade Republic Account",
				Platform:    "traderepublic",
				Credentials: "encrypted_credentials",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			account: Account{
				Platform:    "traderepublic",
				Credentials: "encrypted_credentials",
			},
			wantErr: true,
		},
		{
			name: "invalid platform",
			account: Account{
				Name:        "My Account",
				Platform:    "invalid_platform",
				Credentials: "encrypted_credentials",
			},
			wantErr: true,
		},
		{
			name: "missing credentials",
			account: Account{
				Name:     "My Account",
				Platform: "traderepublic",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Account.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssetValidation(t *testing.T) {
	tests := []struct {
		name    string
		asset   Asset
		wantErr bool
	}{
		{
			name: "valid asset",
			asset: Asset{
				ISIN:     "US0378331005",
				Name:     "Apple Inc.",
				Symbol:   stringPtr("AAPL"),
				Type:     "stock",
				Currency: "USD",
			},
			wantErr: false,
		},
		{
			name: "invalid ISIN format",
			asset: Asset{
				ISIN:     "INVALID",
				Name:     "Apple Inc.",
				Symbol:   stringPtr("AAPL"),
				Type:     "stock",
				Currency: "USD",
			},
			wantErr: true,
		},
		{
			name: "invalid asset type",
			asset: Asset{
				ISIN:     "US0378331005",
				Name:     "Apple Inc.",
				Symbol:   stringPtr("AAPL"),
				Type:     "invalid_type",
				Currency: "USD",
			},
			wantErr: true,
		},
		{
			name: "invalid currency format",
			asset: Asset{
				ISIN:     "US0378331005",
				Name:     "Apple Inc.",
				Symbol:   stringPtr("AAPL"),
				Type:     "stock",
				Currency: "us",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.asset.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Asset.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssetPriceValidation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		price   AssetPrice
		wantErr bool
	}{
		{
			name: "valid price",
			price: AssetPrice{
				ISIN:      "US0378331005",
				Price:     150.25,
				Currency:  "USD",
				Timestamp: now,
			},
			wantErr: false,
		},
		{
			name: "missing ISIN",
			price: AssetPrice{
				Price:     150.25,
				Currency:  "USD",
				Timestamp: now,
			},
			wantErr: true,
		},
		{
			name: "invalid price",
			price: AssetPrice{
				ISIN:      "US0378331005",
				Price:     -10.0,
				Currency:  "USD",
				Timestamp: now,
			},
			wantErr: true,
		},
		{
			name: "missing timestamp",
			price: AssetPrice{
				ISIN:     "US0378331005",
				Price:    150.25,
				Currency: "USD",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.price.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AssetPrice.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransactionValidation(t *testing.T) {
	tests := []struct {
		name        string
		transaction Transaction
		wantErr     bool
	}{
		{
			name: "valid transaction",
			transaction: Transaction{
				ID:             "txn_123",
				AccountID:      "acc_123",
				Timestamp:      time.Now().Format(time.RFC3339),
				AmountCurrency: "EUR",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			transaction: Transaction{
				AccountID:      "acc_123",
				Timestamp:      time.Now().Format(time.RFC3339),
				AmountCurrency: "EUR",
			},
			wantErr: true,
		},
		{
			name: "missing account ID",
			transaction: Transaction{
				ID:             "txn_123",
				Timestamp:      time.Now().Format(time.RFC3339),
				AmountCurrency: "EUR",
			},
			wantErr: true,
		},
		{
			name: "invalid timestamp format",
			transaction: Transaction{
				ID:             "txn_123",
				AccountID:      "acc_123",
				Timestamp:      "invalid_timestamp",
				AmountCurrency: "EUR",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transaction.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Transaction.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
