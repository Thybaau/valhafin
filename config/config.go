package config

import (
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Secret   SecretConfig   `mapstructure:"secret"`
	General  GeneralConfig  `mapstructure:"general"`
	Database DatabaseConfig `mapstructure:"database"`
	Server   ServerConfig   `mapstructure:"server"`
}

type SecretConfig struct {
	PhoneNumber string `mapstructure:"phone_number"`
	Pin         string `mapstructure:"pin"`
}

type GeneralConfig struct {
	OutputFormat   string `mapstructure:"output_format"`
	OutputFolder   string `mapstructure:"output_folder"`
	ExtractDetails bool   `mapstructure:"extract_details"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

type ServerConfig struct {
	Port          string `mapstructure:"port"`
	EncryptionKey string `mapstructure:"encryption_key"`
}

func Load() (*Config, error) {
	// Try to load from config.yaml first (for backward compatibility)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// Read config file if it exists (ignore error if not found)
	_ = viper.ReadInConfig()

	// Bind environment variables
	viper.AutomaticEnv()
	viper.BindEnv("database.url", "DATABASE_URL")
	viper.BindEnv("server.port", "PORT")
	viper.BindEnv("server.encryption_key", "ENCRYPTION_KEY")

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("general.output_format", "json")
	viper.SetDefault("general.output_folder", "out")
	viper.SetDefault("general.extract_details", false)

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Override with environment variables if set
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		config.Database.URL = dbURL
	}
	if port := os.Getenv("PORT"); port != "" {
		config.Server.Port = port
	}
	if encKey := os.Getenv("ENCRYPTION_KEY"); encKey != "" {
		config.Server.EncryptionKey = encKey
	}

	return &config, nil
}
