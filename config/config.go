package config

import (
	"fmt"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// LoadConfig loads the configuration from a .env file and config.yaml file.
// It first loads environment variables from .env file, then loads and unmarshals
// configuration from config.yaml into the provided type T.
func LoadConfig[T any](path string) (*T, error) {
	// Load .env file if it exists
	envPath := filepath.Join(path, ".env")
	_ = godotenv.Load(envPath)

	// Configure viper
	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal config into struct
	var config T
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}
