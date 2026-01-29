package database

import (
	"fmt"
	"time"
	"valhafin/internal/domain/models"

	"github.com/google/uuid"
)

// CreateAccount creates a new account in the database
func (db *DB) CreateAccount(account *models.Account) error {
	// Generate UUID if not provided
	if account.ID == "" {
		account.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now

	// Validate account
	if err := account.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO accounts (id, name, platform, credentials, created_at, updated_at, last_sync)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := db.Exec(
		query,
		account.ID,
		account.Name,
		account.Platform,
		account.Credentials,
		account.CreatedAt,
		account.UpdatedAt,
		account.LastSync,
	)

	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	return nil
}

// GetAccountByID retrieves an account by its ID
func (db *DB) GetAccountByID(id string) (*models.Account, error) {
	var account models.Account

	query := `
		SELECT id, name, platform, credentials, created_at, updated_at, last_sync
		FROM accounts
		WHERE id = $1
	`

	err := db.Get(&account, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}

// GetAllAccounts retrieves all accounts
func (db *DB) GetAllAccounts() ([]models.Account, error) {
	var accounts []models.Account

	query := `
		SELECT id, name, platform, credentials, created_at, updated_at, last_sync
		FROM accounts
		ORDER BY created_at DESC
	`

	err := db.Select(&accounts, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	return accounts, nil
}

// GetAccountsByPlatform retrieves all accounts for a specific platform
func (db *DB) GetAccountsByPlatform(platform string) ([]models.Account, error) {
	var accounts []models.Account

	query := `
		SELECT id, name, platform, credentials, created_at, updated_at, last_sync
		FROM accounts
		WHERE platform = $1
		ORDER BY created_at DESC
	`

	err := db.Select(&accounts, query, platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts by platform: %w", err)
	}

	return accounts, nil
}

// UpdateAccount updates an existing account
func (db *DB) UpdateAccount(account *models.Account) error {
	// Update timestamp
	account.UpdatedAt = time.Now()

	// Validate account
	if err := account.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE accounts
		SET name = $1, platform = $2, credentials = $3, updated_at = $4, last_sync = $5
		WHERE id = $6
	`

	result, err := db.Exec(
		query,
		account.Name,
		account.Platform,
		account.Credentials,
		account.UpdatedAt,
		account.LastSync,
		account.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("account not found")
	}

	return nil
}

// UpdateAccountLastSync updates the last_sync timestamp for an account
func (db *DB) UpdateAccountLastSync(accountID string, lastSync time.Time) error {
	query := `
		UPDATE accounts
		SET last_sync = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := db.Exec(query, lastSync, time.Now(), accountID)
	if err != nil {
		return fmt.Errorf("failed to update last sync: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("account not found")
	}

	return nil
}

// DeleteAccount deletes an account and all associated transactions (cascade)
func (db *DB) DeleteAccount(id string) error {
	query := `DELETE FROM accounts WHERE id = $1`

	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("account not found")
	}

	return nil
}
