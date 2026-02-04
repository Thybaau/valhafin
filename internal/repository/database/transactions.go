package database

import (
	"encoding/json"
	"fmt"
	"valhafin/internal/domain/models"
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

	// Ensure the asset exists if ISIN is provided
	// Convert empty ISIN to NULL for database
	var isinValue interface{}
	if transaction.ISIN != nil && *transaction.ISIN != "" {
		isinValue = *transaction.ISIN

		// Extract symbol and name from metadata if available
		var symbol *string
		var assetName string = "Unknown"
		if transaction.Metadata != nil && *transaction.Metadata != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(*transaction.Metadata), &metadata); err == nil {
				if symbolStr, ok := metadata["symbol"].(string); ok && symbolStr != "" {
					symbol = &symbolStr
				}
				if nameStr, ok := metadata["name"].(string); ok && nameStr != "" {
					assetName = nameStr
				}
			}
		}

		// Fallback to transaction title if name not in metadata
		if assetName == "Unknown" && transaction.Title != "" {
			assetName = transaction.Title
		}

		// Create asset if it doesn't exist, or update symbol and name if provided
		_, err := db.Exec(`
			INSERT INTO assets (isin, name, symbol, type, currency)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (isin) DO UPDATE
			SET symbol = COALESCE(EXCLUDED.symbol, assets.symbol),
			    name = CASE WHEN assets.name = 'Unknown' THEN EXCLUDED.name ELSE assets.name END
		`, *transaction.ISIN, assetName, symbol, "stock", "EUR")
		if err != nil {
			return fmt.Errorf("failed to create asset for ISIN %s: %w", *transaction.ISIN, err)
		}
	} else {
		isinValue = nil
	}

	tableName := getTransactionTableName(platform)

	// Handle metadata - convert empty string to NULL for JSONB
	var metadata *string
	if transaction.Metadata != nil && *transaction.Metadata != "" {
		metadata = transaction.Metadata
	}

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
		ON CONFLICT (id) DO UPDATE SET
			shares = EXCLUDED.shares,
			share_price = EXCLUDED.share_price,
			quantity = EXCLUDED.quantity,
			fees = EXCLUDED.fees
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
		isinValue, // Use isinValue instead of transaction.ISIN
		transaction.Quantity,
		transaction.TransactionType,
		metadata,
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

	// First, ensure all ISINs exist in the assets table
	// Also extract symbols and names from transaction metadata
	type assetInfo struct {
		isin   string
		name   string
		symbol *string
	}
	assetsToCreate := make(map[string]assetInfo)

	for _, transaction := range transactions {
		if transaction.ISIN != nil && *transaction.ISIN != "" {
			isin := *transaction.ISIN

			// Extract symbol and name from metadata if available
			var symbol *string
			var assetName string = "Unknown"
			if transaction.Metadata != nil && *transaction.Metadata != "" {
				// Parse metadata JSON to extract symbol and name
				var metadata map[string]interface{}
				if err := json.Unmarshal([]byte(*transaction.Metadata), &metadata); err == nil {
					if symbolStr, ok := metadata["symbol"].(string); ok && symbolStr != "" {
						symbol = &symbolStr
					}
					if nameStr, ok := metadata["name"].(string); ok && nameStr != "" {
						assetName = nameStr
					}
				}
			}

			// Fallback to transaction title if name not in metadata
			if assetName == "Unknown" && transaction.Title != "" {
				assetName = transaction.Title
			}

			// Store asset info (symbol and name will be updated if found in later transactions)
			if existing, exists := assetsToCreate[isin]; !exists || (symbol != nil && existing.symbol == nil) {
				assetsToCreate[isin] = assetInfo{isin: isin, name: assetName, symbol: symbol}
			}
		}
	}

	// Create assets for ISINs that don't exist yet
	for _, info := range assetsToCreate {
		// Try to insert the asset, or update symbol and name if it already exists
		_, err := tx.Exec(`
			INSERT INTO assets (isin, name, symbol, type, currency)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (isin) DO UPDATE
			SET symbol = COALESCE(EXCLUDED.symbol, assets.symbol),
			    name = CASE WHEN assets.name = 'Unknown' THEN EXCLUDED.name ELSE assets.name END
		`, info.isin, info.name, info.symbol, "stock", "EUR")
		if err != nil {
			return fmt.Errorf("failed to create asset for ISIN %s: %w", info.isin, err)
		}
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
		ON CONFLICT (id) DO UPDATE SET
			shares = EXCLUDED.shares,
			share_price = EXCLUDED.share_price,
			quantity = EXCLUDED.quantity,
			fees = EXCLUDED.fees
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

		// Handle metadata - convert empty string to NULL for JSONB
		var metadata *string
		if transaction.Metadata != nil && *transaction.Metadata != "" {
			metadata = transaction.Metadata
		}

		// Handle ISIN - convert empty string to NULL
		var isinValue interface{}
		if transaction.ISIN != nil && *transaction.ISIN != "" {
			isinValue = *transaction.ISIN
		} else {
			isinValue = nil
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
			isinValue, // Use isinValue instead of transaction.ISIN
			transaction.Quantity,
			transaction.TransactionType,
			metadata,
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
		WHERE account_id = $1 AND (subtitle IS NULL OR subtitle != 'Échec du plan d''épargne')
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

// GetTransactionsByAccountWithSort retrieves transactions for a specific account with custom sorting
func (db *DB) GetTransactionsByAccountWithSort(accountID string, platform string, filter TransactionFilter, sortBy, sortOrder string) ([]models.Transaction, error) {
	tableName := getTransactionTableName(platform)

	query := fmt.Sprintf(`
		SELECT 
			t.id, t.account_id, t.timestamp, t.title, t.icon, t.avatar, t.subtitle,
			t.amount_currency, t.amount_value, t.amount_fraction, t.status,
			t.action_type, t.action_payload, t.cash_account_number, t.hidden, t.deleted,
			t.actions, t.dividend_per_share, t.taxes, t.total, t.shares, t.share_price,
			t.fees, t.amount, t.isin, t.quantity, t.transaction_type, t.metadata
		FROM %s t
		LEFT JOIN assets a ON t.isin = a.isin
		WHERE t.account_id = $1 AND (t.subtitle IS NULL OR t.subtitle != 'Échec du plan d''épargne')
	`, tableName)

	args := []interface{}{accountID}
	argCount := 1

	// Apply filters
	if filter.StartDate != "" {
		argCount++
		query += fmt.Sprintf(" AND t.timestamp >= $%d", argCount)
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != "" {
		argCount++
		query += fmt.Sprintf(" AND t.timestamp <= $%d", argCount)
		args = append(args, filter.EndDate)
	}

	if filter.ISIN != "" {
		argCount++
		// Search by ISIN (case-insensitive partial match) OR asset name (case-insensitive partial match)
		query += fmt.Sprintf(" AND (LOWER(t.isin) LIKE LOWER($%d) OR LOWER(a.name) LIKE LOWER($%d))", argCount, argCount)
		args = append(args, "%"+filter.ISIN+"%")
	}

	if filter.TransactionType != "" {
		argCount++
		query += fmt.Sprintf(" AND t.transaction_type = $%d", argCount)
		args = append(args, filter.TransactionType)
	}

	// Apply sorting
	if sortBy == "timestamp" {
		if sortOrder == "asc" {
			query += " ORDER BY t.timestamp ASC"
		} else {
			query += " ORDER BY t.timestamp DESC"
		}
	} else if sortBy == "amount" {
		if sortOrder == "asc" {
			query += " ORDER BY t.amount_value ASC"
		} else {
			query += " ORDER BY t.amount_value DESC"
		}
	} else {
		// Default sort
		query += " ORDER BY t.timestamp DESC"
	}

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
		WHERE (subtitle IS NULL OR subtitle != 'Échec du plan d''épargne')
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

// GetAllTransactionsWithSort retrieves all transactions across all accounts for a platform with custom sorting
func (db *DB) GetAllTransactionsWithSort(platform string, filter TransactionFilter, sortBy, sortOrder string) ([]models.Transaction, error) {
	tableName := getTransactionTableName(platform)

	query := fmt.Sprintf(`
		SELECT 
			t.id, t.account_id, t.timestamp, t.title, t.icon, t.avatar, t.subtitle,
			t.amount_currency, t.amount_value, t.amount_fraction, t.status,
			t.action_type, t.action_payload, t.cash_account_number, t.hidden, t.deleted,
			t.actions, t.dividend_per_share, t.taxes, t.total, t.shares, t.share_price,
			t.fees, t.amount, t.isin, t.quantity, t.transaction_type, t.metadata
		FROM %s t
		LEFT JOIN assets a ON t.isin = a.isin
		WHERE (t.subtitle IS NULL OR t.subtitle != 'Échec du plan d''épargne')
	`, tableName)

	args := []interface{}{}
	argCount := 0

	// Apply filters
	if filter.StartDate != "" {
		argCount++
		query += fmt.Sprintf(" AND t.timestamp >= $%d", argCount)
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != "" {
		argCount++
		query += fmt.Sprintf(" AND t.timestamp <= $%d", argCount)
		args = append(args, filter.EndDate)
	}

	if filter.ISIN != "" {
		argCount++
		// Search by ISIN (case-insensitive partial match) OR asset name (case-insensitive partial match)
		query += fmt.Sprintf(" AND (LOWER(t.isin) LIKE LOWER($%d) OR LOWER(a.name) LIKE LOWER($%d))", argCount, argCount)
		args = append(args, "%"+filter.ISIN+"%")
	}

	if filter.TransactionType != "" {
		argCount++
		query += fmt.Sprintf(" AND t.transaction_type = $%d", argCount)
		args = append(args, filter.TransactionType)
	}

	// Apply sorting
	if sortBy == "timestamp" {
		if sortOrder == "asc" {
			query += " ORDER BY t.timestamp ASC"
		} else {
			query += " ORDER BY t.timestamp DESC"
		}
	} else if sortBy == "amount" {
		if sortOrder == "asc" {
			query += " ORDER BY t.amount_value ASC"
		} else {
			query += " ORDER BY t.amount_value DESC"
		}
	} else {
		// Default sort
		query += " ORDER BY t.timestamp DESC"
	}

	// Don't apply pagination here - let the handler do it for combined results
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

// UpdateTransaction updates an existing transaction
func (db *DB) UpdateTransaction(transaction *models.Transaction, platform string) error {
	// Validate transaction
	if err := transaction.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	tableName := getTransactionTableName(platform)

	// Handle ISIN - convert empty string to NULL
	var isinValue interface{}
	if transaction.ISIN != nil && *transaction.ISIN != "" {
		isinValue = *transaction.ISIN
	} else {
		isinValue = nil
	}

	query := fmt.Sprintf(`
		UPDATE %s SET
			title = $1,
			subtitle = $2,
			amount_value = $3,
			amount_currency = $4,
			fees = $5,
			quantity = $6,
			transaction_type = $7,
			isin = $8
		WHERE id = $9
	`, tableName)

	result, err := db.Exec(
		query,
		transaction.Title,
		transaction.Subtitle,
		transaction.AmountValue,
		transaction.AmountCurrency,
		transaction.Fees,
		transaction.Quantity,
		transaction.TransactionType,
		isinValue,
		transaction.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
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

	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s t
		LEFT JOIN assets a ON t.isin = a.isin
		WHERE (t.subtitle IS NULL OR t.subtitle != 'Échec du plan d''épargne')
	`, tableName)

	args := []interface{}{}
	argCount := 0

	if filter.AccountID != "" {
		argCount++
		query += fmt.Sprintf(" AND t.account_id = $%d", argCount)
		args = append(args, filter.AccountID)
	}

	if filter.StartDate != "" {
		argCount++
		query += fmt.Sprintf(" AND t.timestamp >= $%d", argCount)
		args = append(args, filter.StartDate)
	}

	if filter.EndDate != "" {
		argCount++
		query += fmt.Sprintf(" AND t.timestamp <= $%d", argCount)
		args = append(args, filter.EndDate)
	}

	if filter.ISIN != "" {
		argCount++
		// Search by ISIN (case-insensitive partial match) OR asset name (case-insensitive partial match)
		query += fmt.Sprintf(" AND (LOWER(t.isin) LIKE LOWER($%d) OR LOWER(a.name) LIKE LOWER($%d))", argCount, argCount)
		args = append(args, "%"+filter.ISIN+"%")
	}

	if filter.TransactionType != "" {
		argCount++
		query += fmt.Sprintf(" AND t.transaction_type = $%d", argCount)
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
