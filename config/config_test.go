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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	t.Run("LoadConfig", func(t *testing.T) {
		// Create a dummy .env file
		file, err := os.Create(".env")
		assert.NoError(t, err)
		defer os.Remove(".env")

		_, err = file.WriteString("JWT_SECRET=test_secret\nLOG_LEVEL=debug")
		assert.NoError(t, err)
		file.Close()

		type Config struct {
			JWTSecret string `mapstructure:"JWT_SECRET"`
			LogLevel  string `mapstructure:"LOG_LEVEL"`
		}

		config, err := LoadConfig[Config]()
		assert.NoError(t, err)
		assert.Equal(t, "test_secret", config.JWTSecret)
		assert.Equal(t, "debug", config.LogLevel)
	})
}