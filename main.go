package main

import (
	"fmt"
	"log"
	"os"

	"valhafin/config"
	"valhafin/scrapers/traderepublic"
	"valhafin/utils"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(cfg.General.OutputFolder, 0755); err != nil {
		log.Fatalf("❌ Failed to create output directory: %v", err)
	}

	// Initialize Trade Republic scraper
	tr := traderepublic.NewScraper(cfg)

	// Authenticate
	fmt.Println("🔐 Authenticating with Trade Republic...")
	sessionToken, err := tr.Authenticate()
	if err != nil {
		log.Fatalf("❌ Authentication failed: %v", err)
	}

	fmt.Println("✅ Authentication successful!")

	// Fetch transactions
	fmt.Println("📊 Fetching transactions...")
	transactions, err := tr.FetchTransactions(sessionToken)
	if err != nil {
		log.Fatalf("❌ Failed to fetch transactions: %v", err)
	}

	// Export transactions
	transactionsFile := cfg.General.OutputFolder + "/trade_republic_transactions." + cfg.General.OutputFormat
	if err := utils.ExportData(transactions, transactionsFile, cfg.General.OutputFormat); err != nil {
		log.Fatalf("❌ Failed to export transactions: %v", err)
	}

	fmt.Printf("✅ Transactions saved to '%s'\n", transactionsFile)

	// Fetch profile cash
	fmt.Println("💰 Fetching profile cash...")
	profileCash, err := tr.FetchProfileCash(sessionToken)
	if err != nil {
		log.Fatalf("❌ Failed to fetch profile cash: %v", err)
	}

	// Export profile cash
	profileCashFile := cfg.General.OutputFolder + "/trade_republic_profile_cash." + cfg.General.OutputFormat
	if err := utils.ExportData(profileCash, profileCashFile, cfg.General.OutputFormat); err != nil {
		log.Fatalf("❌ Failed to export profile cash: %v", err)
	}

	fmt.Printf("✅ Profile cash saved to '%s'\n", profileCashFile)
	fmt.Println("\n🎉 All data successfully exported!")
}
