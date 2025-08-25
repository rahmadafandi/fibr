// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
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

		// Set field value based on type
		fieldValue := v.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(envValue)
		case reflect.Int:
			if intVal, err := strconv.Atoi(envValue); err == nil {
				fieldValue.SetInt(int64(intVal))
			}
		case reflect.Bool:
			if boolVal, err := strconv.ParseBool(envValue); err == nil {
				fieldValue.SetBool(boolVal)
			}
		}
	}

	return &config, nil
}