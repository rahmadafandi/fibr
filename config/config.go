package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/joho/godotenv"
)

// LoadConfig loads the configuration from a .env file.
// It loads environment variables from .env file and maps them to the provided type T
// based on the "mapstructure" struct tags.
func LoadConfig[T any]() (*T, error) {
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting working directory: %w", err)
	}

	// Load .env file if it exists
	envPath := filepath.Join(wd, ".env")
	if err := godotenv.Load(envPath); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	// Create new config instance
	var config T
	v := reflect.ValueOf(&config).Elem()
	t := v.Type()

	// Iterate through struct fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			continue
		}

		// Get env value
		envValue := os.Getenv(strings.ToUpper(tag))
		if envValue == "" {
			continue
		}

		// Set field value
		fieldValue := v.Field(i)
		if fieldValue.CanSet() {
			fieldValue.SetString(envValue)
		}
	}

	return &config, nil
}
