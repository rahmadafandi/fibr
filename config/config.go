package config

import (
	"github.com/spf13/viper"
)

// Config holds the configuration for the application.
type Config struct {
	JWTSecret string `mapstructure:"JWT_SECRET"`
	LogLevel  string `mapstructure:"LOG_LEVEL"`
}

// LoadConfig loads the configuration from a .env file or environment variables.
func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
