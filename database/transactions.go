package database

import (
	"encoding/json"
	"fmt"
	"valhafin/models"
)

// TransactionFilter holds filter parameters for querying transactions
type TransactionFilter struct {
	AccountID       string
	StartDate       string
	EndDate         string
	ISIN            string
	TransactionType string
	Page            int
	Limit           int
}

// CreateTransaction creates a new transaction in the appropriate platform table
func (db *DB) CreateTransaction(transaction *models.Transaction, platform string) error {
	// Validate transaction
	if err := transaction.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	tableName := getTransactionTableName(platform)

	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, account_id, timestamp, title, icon, avatar, subtitle,
			amount_currency, amount_value, amount_fraction, status,
			action_type, action_payload, cash_account_number, hidden, deleted,
			actions, dividend_per_share, taxes, total, shares, share_price,
			fees, amount, isin, quantity, transaction_type, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28
		)
		ON CONFLICT (id) DO NOTHING
	`, tableName)

	_, err := db.Exec(
		query,
		transaction.ID,
		transaction.AccountID,
		transaction.Timestamp,
		transaction.Title,
		transaction.Icon,
		transaction.Avatar,
		transaction.Subtitle,
		transaction.AmountCurrency,
		transaction.AmountValue,
		transaction.AmountFraction,
		transaction.Status,
		transaction.ActionType,
		transaction.ActionPayload,
		transaction.CashAccountNumber,
		transaction.Hidden,
		transaction.Deleted,
		transaction.Actions,
		transaction.DividendPerShare,
		transaction.Taxes,
		transaction.Total,
		transaction.Shares,
		transaction.SharePrice,
		transaction.Fees,
		transaction.Amount,
		transaction.ISIN,
		transaction.Quantity,
		transaction.TransactionType,
		transaction.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// CreateTransactionsBatch creates multiple transactions in a single transaction
func (db *DB) CreateTransactionsBatch(transactions []models.Transaction, platform string) error {
	if len(transactions) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	tableName := getTransactionTableName(platform)

	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, account_id, timestamp, title, icon, avatar, subtitle,
			amount_currency, amount_value, amount_fraction, status,
			action_type, action_payload, cash_account_number, hidden, deleted,
			actions, dividend_per_share, taxes, total, shares, share_price,
			fees, amount, isin, quantity, transaction_type, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28
		)
		ON CONFLICT (id) DO NOTHING
	`, tableName)

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, transaction := range transactions {
		if err := transaction.Validate(); err != nil {
			return fmt.Errorf("validation failed for transaction %s: %w", transaction.ID, err)
		}

		_, err := stmt.Exec(
			transaction.ID,
			transaction.AccountID,
			transaction.Timestamp,
			transaction.Title,
			transaction.Icon,
			transaction.Avatar,
			transaction.Subtitle,
			transaction.AmountCurrency,
			transaction.AmountValue,
			transaction.AmountFraction,
			transaction.Status,
			transaction.ActionType,
			transaction.ActionPayload,
			transaction.CashAccountNumber,
			transaction.Hidden,
			transaction.Deleted,
			transaction.Actions,
			transaction.DividendPerShare,
			transaction.Taxes,
			transaction.Total,
			transaction.Shares,
			transaction.SharePrice,
			transaction.Fees,
			transaction.Amount,
			transaction.ISIN,
			transaction.Quantity,
			transaction.TransactionType,
			transaction.Metadata,
		)

		if err != nil {
			return fmt.Errorf("failed to insert transaction %s: %w", transaction.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetTransactionsByAccount retrieves all transactions for a specific account
func (db *DB) GetTransactionsByAccount(accountID string, platform string, filter TransactionFilter) ([]models.Transaction, error) {
	tableName := getTransactionTableName(platform)

	query := fmt.Sprintf(`
		SELECT 
			id, account_id, timestamp, title, icon, avatar, subtitle,
			amount_currency, amount_value, amount_fraction, status,
			action_type, action_payload, cash_account_number, hidden, deleted,
			actions, dividend_per_share, taxes, total, shares, share_price,
			fees, amount, isin, quantity, transaction_type, metadata
		FROM %s
		WHERE account_id = $1
	`, tableName)

	args := []interface{}{accountID}
	argCount := 1

	// Apply filters
	if filter.StartDate != "" {
		argCount++
		query += fmt.Sprintf(" AND timestamp >= $%d", argCount)
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != "" {
		argCount++
		query += fmt.Sprintf(" AND timestamp <= $%d", argCount)
		args = append(args, filter.EndDate)
	}

	if filter.ISIN != "" {
		argCount++
		query += fmt.Sprintf(" AND isin = $%d", argCount)
		args = append(args, filter.ISIN)
	}

	if filter.TransactionType != "" {
		argCount++
		query += fmt.Sprintf(" AND transaction_type = $%d", argCount)
		args = append(args, filter.TransactionType)
	}

	query += " ORDER BY timestamp DESC"

	// Apply pagination
	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)

		if filter.Page > 0 {
			argCount++
			offset := (filter.Page - 1) * filter.Limit
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, offset)
		}
	}

	var transactions []models.Transaction
	err := db.Select(&transactions, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	return transactions, nil
}

// GetAllTransactions retrieves all transactions across all accounts for a platform
func (db *DB) GetAllTransactions(platform string, filter TransactionFilter) ([]models.Transaction, error) {
	tableName := getTransactionTableName(platform)

	query := fmt.Sprintf(`
		SELECT 
			id, account_id, timestamp, title, icon, avatar, subtitle,
			amount_currency, amount_value, amount_fraction, status,
			action_type, action_payload, cash_account_number, hidden, deleted,
			actions, dividend_per_share, taxes, total, shares, share_price,
			fees, amount, isin, quantity, transaction_type, metadata
		FROM %s
		WHERE 1=1
	`, tableName)

	args := []interface{}{}
	argCount := 0

	// Apply filters
	if filter.StartDate != "" {
		argCount++
		query += fmt.Sprintf(" AND timestamp >= $%d", argCount)
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != "" {
		argCount++
		query += fmt.Sprintf(" AND timestamp <= $%d", argCount)
		args = append(args, filter.EndDate)
	}

	if filter.ISIN != "" {
		argCount++
		query += fmt.Sprintf(" AND isin = $%d", argCount)
		args = append(args, filter.ISIN)
	}

	if filter.TransactionType != "" {
		argCount++
		query += fmt.Sprintf(" AND transaction_type = $%d", argCount)
		args = append(args, filter.TransactionType)
	}

	query += " ORDER BY timestamp DESC"

	// Apply pagination
	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)

		if filter.Page > 0 {
			argCount++
			offset := (filter.Page - 1) * filter.Limit
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, offset)
		}
	}

	var transactions []models.Transaction
	err := db.Select(&transactions, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	return transactions, nil
}

// GetTransactionByID retrieves a specific transaction by ID
func (db *DB) GetTransactionByID(id string, platform string) (*models.Transaction, error) {
	tableName := getTransactionTableName(platform)

	query := fmt.Sprintf(`
		SELECT 
			id, account_id, timestamp, title, icon, avatar, subtitle,
			amount_currency, amount_value, amount_fraction, status,
			action_type, action_payload, cash_account_number, hidden, deleted,
			actions, dividend_per_share, taxes, total, shares, share_price,
			fees, amount, isin, quantity, transaction_type, metadata
		FROM %s
		WHERE id = $1
	`, tableName)

	var transaction models.Transaction
	err := db.Get(&transaction, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &transaction, nil
}

// DeleteTransaction deletes a transaction
func (db *DB) DeleteTransaction(id string, platform string) error {
	tableName := getTransactionTableName(platform)

	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, tableName)

	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found")
	}

	return nil
}

// CountTransactions counts transactions matching the filter
func (db *DB) CountTransactions(platform string, filter TransactionFilter) (int, error) {
	tableName := getTransactionTableName(platform)

	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE 1=1`, tableName)

	args := []interface{}{}
	argCount := 0

	if filter.AccountID != "" {
		argCount++
		query += fmt.Sprintf(" AND account_id = $%d", argCount)
		args = append(args, filter.AccountID)
	}

	if filter.StartDate != "" {
		argCount++
		query += fmt.Sprintf(" AND timestamp >= $%d", argCount)
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != "" {
		argCount++
		query += fmt.Sprintf(" AND timestamp <= $%d", argCount)
		args = append(args, filter.EndDate)
	}

	if filter.ISIN != "" {
		argCount++
		query += fmt.Sprintf(" AND isin = $%d", argCount)
		args = append(args, filter.ISIN)
	}

	if filter.TransactionType != "" {
		argCount++
		query += fmt.Sprintf(" AND transaction_type = $%d", argCount)
		args = append(args, filter.TransactionType)
	}

	var count int
	err := db.Get(&count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}

// ImportTransactionsFromJSON imports transactions from JSON data
func (db *DB) ImportTransactionsFromJSON(jsonData []byte, accountID string, platform string) (int, error) {
	var transactions []models.Transaction
	if err := json.Unmarshal(jsonData, &transactions); err != nil {
		return 0, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Set account ID for all transactions
	for i := range transactions {
		transactions[i].AccountID = accountID
	}

	if err := db.CreateTransactionsBatch(transactions, platform); err != nil {
		return 0, err
	}

	return len(transactions), nil
}

// getTransactionTableName returns the table name for a given platform
func getTransactionTableName(platform string) string {
	switch platform {
	case "traderepublic":
		return "transactions_traderepublic"
	case "binance":
		return "transactions_binance"
	case "boursedirect":
		return "transactions_boursedirect"
	default:
		return "transactions_traderepublic" // default fallback
	}
}
