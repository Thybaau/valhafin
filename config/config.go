package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Secret  SecretConfig  `mapstructure:"secret"`
	General GeneralConfig `mapstructure:"general"`
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

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
