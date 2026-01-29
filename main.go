package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"

	"valhafin/internal/api"
	"valhafin/internal/config"
	"valhafin/internal/repository/database"
	encryptionsvc "valhafin/internal/service/encryption"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Parse database URL
	dbConfig, err := parseDatabaseURL(cfg.Database.URL)
	if err != nil {
		log.Fatalf("❌ Failed to parse database URL: %v", err)
	}

	// Connect to database
	db, err := database.Connect(dbConfig)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}

	// Initialize encryption service
	encryptionKey, err := getEncryptionKey(cfg.Server.EncryptionKey)
	if err != nil {
		log.Fatalf("❌ Failed to get encryption key: %v", err)
	}

	encryptionService, err := encryptionsvc.NewEncryptionService(encryptionKey)
	if err != nil {
		log.Fatalf("❌ Failed to initialize encryption service: %v", err)
	}

	// Setup routes
	router := api.SetupRoutes(db, encryptionService)

	// Start server
	port := cfg.Server.Port
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("🚀 Server starting on %s", addr)
	log.Printf("📊 API available at http://localhost%s/api", addr)
	log.Printf("💚 Health check at http://localhost%s/health", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}

// parseDatabaseURL parses a PostgreSQL connection URL
func parseDatabaseURL(url string) (database.Config, error) {
	// Example: postgresql://user:password@localhost:5432/dbname?sslmode=disable
	cfg := database.Config{
		SSLMode: "disable",
	}

	if url == "" {
		return cfg, fmt.Errorf("database URL is empty")
	}

	// Remove postgresql:// prefix
	url = strings.TrimPrefix(url, "postgresql://")
	url = strings.TrimPrefix(url, "postgres://")

	// Split by @
	parts := strings.Split(url, "@")
	if len(parts) != 2 {
		return cfg, fmt.Errorf("invalid database URL format")
	}

	// Parse user:password
	userPass := strings.Split(parts[0], ":")
	if len(userPass) == 2 {
		cfg.User = userPass[0]
		cfg.Password = userPass[1]
	}

	// Parse host:port/dbname
	hostParts := strings.Split(parts[1], "/")
	if len(hostParts) != 2 {
		return cfg, fmt.Errorf("invalid database URL format")
	}

	// Parse host:port
	hostPort := strings.Split(hostParts[0], ":")
	cfg.Host = hostPort[0]
	if len(hostPort) == 2 {
		var port int
		fmt.Sscanf(hostPort[1], "%d", &port)
		cfg.Port = port
	} else {
		cfg.Port = 5432
	}

	// Parse dbname and query params
	dbParts := strings.Split(hostParts[1], "?")
	cfg.DBName = dbParts[0]

	// Parse query params
	if len(dbParts) == 2 {
		params := strings.Split(dbParts[1], "&")
		for _, param := range params {
			kv := strings.Split(param, "=")
			if len(kv) == 2 && kv[0] == "sslmode" {
				cfg.SSLMode = kv[1]
			}
		}
	}

	return cfg, nil
}

// getEncryptionKey gets the encryption key from config or environment
func getEncryptionKey(keyStr string) ([]byte, error) {
	if keyStr == "" {
		return nil, fmt.Errorf("encryption key is not set (set ENCRYPTION_KEY environment variable)")
	}

	// Try to decode as hex
	key, err := hex.DecodeString(keyStr)
	if err == nil && len(key) == 32 {
		return key, nil
	}

	// If not hex, use the string directly (must be 32 bytes)
	keyBytes := []byte(keyStr)
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes (got %d bytes)", len(keyBytes))
	}

	return keyBytes, nil
}
