package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// LoadConfig loads the configuration from a .env file or environment variables.
func LoadConfig[T any](path string) (*T, error) {
	_ = godotenv.Load(fmt.Sprintf("%s/.env", path))
	viper.AddConfigPath(path)
	viper.SetConfigName(".env.yaml")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config T
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
