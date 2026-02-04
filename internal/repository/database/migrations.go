package database

import (
	"fmt"
	"log"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	Up      string
	Down    string
}

// migrations holds all database migrations in order
var migrations = []Migration{
	{
		Version: 1,
		Name:    "create_accounts_table",
		Up: `
			CREATE TABLE IF NOT EXISTS accounts (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				name VARCHAR(255) NOT NULL,
				platform VARCHAR(50) NOT NULL,
				credentials TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				last_sync TIMESTAMP
			);
			
			CREATE INDEX IF NOT EXISTS idx_accounts_platform ON accounts(platform);
		`,
		Down: `
			DROP TABLE IF EXISTS accounts CASCADE;
		`,
	},
	{
		Version: 2,
		Name:    "create_assets_table",
		Up: `
			CREATE TABLE IF NOT EXISTS assets (
				isin VARCHAR(12) PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				symbol VARCHAR(20),
				type VARCHAR(20) NOT NULL,
				currency VARCHAR(3) NOT NULL,
				last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
			
			CREATE INDEX IF NOT EXISTS idx_assets_type ON assets(type);
			CREATE INDEX IF NOT EXISTS idx_assets_symbol ON assets(symbol);
		`,
		Down: `
			DROP TABLE IF EXISTS assets CASCADE;
		`,
	},
	{
		Version: 3,
		Name:    "create_asset_prices_table",
		Up: `
			CREATE TABLE IF NOT EXISTS asset_prices (
				id BIGSERIAL PRIMARY KEY,
				isin VARCHAR(12) REFERENCES assets(isin) ON DELETE CASCADE,
				price DECIMAL(20, 8) NOT NULL,
				currency VARCHAR(3) NOT NULL,
				timestamp TIMESTAMP NOT NULL,
				UNIQUE(isin, timestamp)
			);
			
			CREATE INDEX IF NOT EXISTS idx_asset_prices_isin_timestamp ON asset_prices(isin, timestamp DESC);
		`,
		Down: `
			DROP TABLE IF EXISTS asset_prices CASCADE;
		`,
	},
	{
		Version: 4,
		Name:    "create_transactions_traderepublic_table",
		Up: `
			CREATE TABLE IF NOT EXISTS transactions_traderepublic (
				id VARCHAR(255) PRIMARY KEY,
				account_id UUID REFERENCES accounts(id) ON DELETE CASCADE,
				timestamp VARCHAR(255) NOT NULL,
				title VARCHAR(255),
				icon VARCHAR(255),
				avatar VARCHAR(255),
				subtitle VARCHAR(255),
				amount_currency VARCHAR(3),
				amount_value DECIMAL(20, 8),
				amount_fraction INT,
				status VARCHAR(50),
				action_type VARCHAR(50),
				action_payload TEXT,
				cash_account_number VARCHAR(255),
				hidden BOOLEAN DEFAULT FALSE,
				deleted BOOLEAN DEFAULT FALSE,
				actions TEXT,
				dividend_per_share VARCHAR(255),
				taxes VARCHAR(255),
				total VARCHAR(255),
				shares VARCHAR(255),
				share_price VARCHAR(255),
				fees VARCHAR(255),
				amount VARCHAR(255),
				isin VARCHAR(12) REFERENCES assets(isin),
				quantity DECIMAL(20, 8),
				transaction_type VARCHAR(50),
				metadata JSONB
			);
			
			CREATE INDEX IF NOT EXISTS idx_transactions_tr_account ON transactions_traderepublic(account_id);
			CREATE INDEX IF NOT EXISTS idx_transactions_tr_timestamp ON transactions_traderepublic(timestamp DESC);
			CREATE INDEX IF NOT EXISTS idx_transactions_tr_isin ON transactions_traderepublic(isin);
			CREATE INDEX IF NOT EXISTS idx_transactions_tr_type ON transactions_traderepublic(transaction_type);
		`,
		Down: `
			DROP TABLE IF EXISTS transactions_traderepublic CASCADE;
		`,
	},
	{
		Version: 5,
		Name:    "create_transactions_binance_table",
		Up: `
			CREATE TABLE IF NOT EXISTS transactions_binance (
				id VARCHAR(255) PRIMARY KEY,
				account_id UUID REFERENCES accounts(id) ON DELETE CASCADE,
				timestamp VARCHAR(255) NOT NULL,
				title VARCHAR(255),
				icon VARCHAR(255),
				avatar VARCHAR(255),
				subtitle VARCHAR(255),
				amount_currency VARCHAR(3),
				amount_value DECIMAL(20, 8),
				amount_fraction INT,
				status VARCHAR(50),
				action_type VARCHAR(50),
				action_payload TEXT,
				cash_account_number VARCHAR(255),
				hidden BOOLEAN DEFAULT FALSE,
				deleted BOOLEAN DEFAULT FALSE,
				actions TEXT,
				dividend_per_share VARCHAR(255),
				taxes VARCHAR(255),
				total VARCHAR(255),
				shares VARCHAR(255),
				share_price VARCHAR(255),
				fees VARCHAR(255),
				amount VARCHAR(255),
				isin VARCHAR(12) REFERENCES assets(isin),
				quantity DECIMAL(20, 8),
				transaction_type VARCHAR(50),
				metadata JSONB
			);
			
			CREATE INDEX IF NOT EXISTS idx_transactions_bn_account ON transactions_binance(account_id);
			CREATE INDEX IF NOT EXISTS idx_transactions_bn_timestamp ON transactions_binance(timestamp DESC);
			CREATE INDEX IF NOT EXISTS idx_transactions_bn_isin ON transactions_binance(isin);
			CREATE INDEX IF NOT EXISTS idx_transactions_bn_type ON transactions_binance(transaction_type);
		`,
		Down: `
			DROP TABLE IF EXISTS transactions_binance CASCADE;
		`,
	},
	{
		Version: 6,
		Name:    "create_transactions_boursedirect_table",
		Up: `
			CREATE TABLE IF NOT EXISTS transactions_boursedirect (
				id VARCHAR(255) PRIMARY KEY,
				account_id UUID REFERENCES accounts(id) ON DELETE CASCADE,
				timestamp VARCHAR(255) NOT NULL,
				title VARCHAR(255),
				icon VARCHAR(255),
				avatar VARCHAR(255),
				subtitle VARCHAR(255),
				amount_currency VARCHAR(3),
				amount_value DECIMAL(20, 8),
				amount_fraction INT,
				status VARCHAR(50),
				action_type VARCHAR(50),
				action_payload TEXT,
				cash_account_number VARCHAR(255),
				hidden BOOLEAN DEFAULT FALSE,
				deleted BOOLEAN DEFAULT FALSE,
				actions TEXT,
				dividend_per_share VARCHAR(255),
				taxes VARCHAR(255),
				total VARCHAR(255),
				shares VARCHAR(255),
				share_price VARCHAR(255),
				fees VARCHAR(255),
				amount VARCHAR(255),
				isin VARCHAR(12) REFERENCES assets(isin),
				quantity DECIMAL(20, 8),
				transaction_type VARCHAR(50),
				metadata JSONB
			);
			
			CREATE INDEX IF NOT EXISTS idx_transactions_bd_account ON transactions_boursedirect(account_id);
			CREATE INDEX IF NOT EXISTS idx_transactions_bd_timestamp ON transactions_boursedirect(timestamp DESC);
			CREATE INDEX IF NOT EXISTS idx_transactions_bd_isin ON transactions_boursedirect(isin);
			CREATE INDEX IF NOT EXISTS idx_transactions_bd_type ON transactions_boursedirect(transaction_type);
		`,
		Down: `
			DROP TABLE IF EXISTS transactions_boursedirect CASCADE;
		`,
	},
	{
		Version: 7,
		Name:    "create_migrations_table",
		Up: `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`,
		Down: `
			DROP TABLE IF EXISTS schema_migrations CASCADE;
		`,
	},
	{
		Version: 8,
		Name:    "add_symbol_verified_to_assets",
		Up: `
			ALTER TABLE assets ADD COLUMN IF NOT EXISTS symbol_verified BOOLEAN DEFAULT FALSE;
			CREATE INDEX IF NOT EXISTS idx_assets_symbol_verified ON assets(symbol_verified);
		`,
		Down: `
			DROP INDEX IF EXISTS idx_assets_symbol_verified;
			ALTER TABLE assets DROP COLUMN IF EXISTS symbol_verified;
		`,
	},
}

// RunMigrations executes all pending migrations
func (db *DB) RunMigrations() error {
	// First, ensure the migrations table exists
	_, err := db.Exec(migrations[len(migrations)-1].Up)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion := 0
	err = db.Get(&currentVersion, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err != nil {
		log.Printf("Warning: Could not get current migration version: %v", err)
	}

	log.Printf("Current database version: %d", currentVersion)

	// Run pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		log.Printf("Running migration %d: %s", migration.Version, migration.Name)

		// Execute migration
		_, err := db.Exec(migration.Up)
		if err != nil {
			return fmt.Errorf("failed to run migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		// Record migration
		_, err = db.Exec(
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
			migration.Version, migration.Name,
		)
		if err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		log.Printf("✅ Migration %d completed: %s", migration.Version, migration.Name)
	}

	log.Println("✅ All migrations completed successfully")
	return nil
}

// RollbackMigration rolls back the last migration
func (db *DB) RollbackMigration() error {
	// Get current version
	currentVersion := 0
	err := db.Get(&currentVersion, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if currentVersion == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	// Find the migration to rollback
	var migrationToRollback *Migration
	for i := range migrations {
		if migrations[i].Version == currentVersion {
			migrationToRollback = &migrations[i]
			break
		}
	}

	if migrationToRollback == nil {
		return fmt.Errorf("migration %d not found", currentVersion)
	}

	log.Printf("Rolling back migration %d: %s", migrationToRollback.Version, migrationToRollback.Name)

	// Execute rollback
	_, err = db.Exec(migrationToRollback.Down)
	if err != nil {
		return fmt.Errorf("failed to rollback migration %d: %w", currentVersion, err)
	}

	// Remove migration record
	_, err = db.Exec("DELETE FROM schema_migrations WHERE version = $1", currentVersion)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	log.Printf("✅ Migration %d rolled back successfully", currentVersion)
	return nil
}
