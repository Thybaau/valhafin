package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
	"valhafin/internal/repository/database"
	"valhafin/internal/service/encryption"
	"valhafin/internal/service/scraper/types"
)

// ScraperFactoryInterface defines the interface for scraper factories
type ScraperFactoryInterface interface {
	GetScraper(platform string) (types.Scraper, error)
}

// Service handles synchronization of transactions from external platforms
type Service struct {
	db             *database.DB
	scraperFactory ScraperFactoryInterface
	encryption     *encryption.EncryptionService
}

// NewService creates a new synchronization service
func NewService(db *database.DB, scraperFactory ScraperFactoryInterface, encryptionService *encryption.EncryptionService) *Service {
	return &Service{
		db:             db,
		scraperFactory: scraperFactory,
		encryption:     encryptionService,
	}
}

// SyncAccount synchronizes transactions for a specific account
func (s *Service) SyncAccount(accountID string) (*types.SyncResult, error) {
	startTime := time.Now()

	result := &types.SyncResult{
		AccountID: accountID,
		StartTime: startTime,
	}

	// Get account from database
	account, err := s.db.GetAccountByID(accountID)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to retrieve account: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, fmt.Errorf("failed to retrieve account: %w", err)
	}

	result.Platform = account.Platform

	// Decrypt credentials
	credentialsJSON, err := s.encryption.Decrypt(account.Credentials)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to decrypt credentials: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		log.Printf("ERROR: Failed to decrypt credentials for account %s: %v", accountID, err)
		return result, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// Parse credentials
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(credentialsJSON), &credentials); err != nil {
		result.Error = fmt.Sprintf("Failed to parse credentials: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		log.Printf("ERROR: Failed to parse credentials for account %s: %v", accountID, err)
		return result, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Get appropriate scraper
	platformScraper, err := s.scraperFactory.GetScraper(account.Platform)
	if err != nil {
		result.Error = fmt.Sprintf("Unsupported platform: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		log.Printf("ERROR: Unsupported platform for account %s: %v", accountID, err)
		return result, fmt.Errorf("unsupported platform: %w", err)
	}

	// Determine sync type
	syncType := "full"
	if account.LastSync != nil {
		syncType = "incremental"
	}
	result.SyncType = syncType

	log.Printf("INFO: Starting %s sync for account %s (platform: %s)", syncType, accountID, account.Platform)

	// Fetch transactions from platform
	transactions, err := platformScraper.FetchTransactions(credentials, account.LastSync)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to fetch transactions: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()

		// Log detailed error information
		if scraperErr, ok := err.(*types.ScraperError); ok {
			log.Printf("ERROR: Scraper error for account %s - Type: %s, Platform: %s, Message: %s, Retry: %v",
				accountID, scraperErr.Type, scraperErr.Platform, scraperErr.Message, scraperErr.Retry)
		} else {
			log.Printf("ERROR: Failed to fetch transactions for account %s: %v", accountID, err)
		}

		return result, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	result.TransactionsFetched = len(transactions)
	log.Printf("INFO: Fetched %d transactions for account %s", len(transactions), accountID)

	// Set account ID for all transactions
	for i := range transactions {
		transactions[i].AccountID = accountID
	}

	// Store transactions in database
	if len(transactions) > 0 {
		if err := s.db.CreateTransactionsBatch(transactions, account.Platform); err != nil {
			result.Error = fmt.Sprintf("Failed to store transactions: %v", err)
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime).String()
			log.Printf("ERROR: Failed to store transactions for account %s: %v", accountID, err)
			return result, fmt.Errorf("failed to store transactions: %w", err)
		}
		result.TransactionsStored = len(transactions)
		log.Printf("INFO: Stored %d transactions for account %s", len(transactions), accountID)
	}

	// Update last sync timestamp
	now := time.Now()
	if err := s.db.UpdateAccountLastSync(accountID, now); err != nil {
		// Log warning but don't fail the sync
		log.Printf("WARNING: Failed to update last sync timestamp for account %s: %v", accountID, err)
	}

	result.EndTime = time.Now()
	result.Duration = time.Since(startTime).String()

	log.Printf("INFO: Sync completed for account %s - Fetched: %d, Stored: %d, Duration: %s",
		accountID, result.TransactionsFetched, result.TransactionsStored, result.Duration)

	return result, nil
}

// SyncAllAccounts synchronizes all accounts (skips Trade Republic for automatic sync)
func (s *Service) SyncAllAccounts() ([]types.SyncResult, error) {
	accounts, err := s.db.GetAllAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve accounts: %w", err)
	}

	results := make([]types.SyncResult, 0, len(accounts))

	for _, account := range accounts {
		// Skip Trade Republic accounts for automatic sync (requires 2FA)
		if account.Platform == "traderepublic" {
			log.Printf("INFO: Skipping automatic sync for Trade Republic account %s (requires 2FA)", account.ID)
			continue
		}

		result, err := s.SyncAccount(account.ID)
		if err != nil {
			// Continue with other accounts even if one fails
			log.Printf("WARNING: Failed to sync account %s: %v", account.ID, err)
		}
		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

// GetScraper returns a scraper for the specified platform
func (s *Service) GetScraper(platform string) types.Scraper {
	scraper, err := s.scraperFactory.GetScraper(platform)
	if err != nil {
		return nil
	}
	return scraper
}
