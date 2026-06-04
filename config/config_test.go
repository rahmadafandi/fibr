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
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	t.Run("BasicTypes", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("JWT_SECRET", "test_secret")
		os.Setenv("LOG_LEVEL", "debug")

		type Config struct {
			JWTSecret string `mapstructure:"JWT_SECRET"`
			LogLevel  string `mapstructure:"LOG_LEVEL"`
		}

		var cfg Config
		err := LoadConfig(&cfg)
		assert.NoError(t, err)
		assert.Equal(t, "test_secret", cfg.JWTSecret)
		assert.Equal(t, "debug", cfg.LogLevel)
	})

	t.Run("ExtendedTypes", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("PORT", "8080")
		os.Setenv("RATE", "1.5")
		os.Setenv("DEBUG", "true")
		os.Setenv("TIMEOUT", "30s")
		os.Setenv("HOSTS", "a.com, b.com ,c.com")
		os.Setenv("WORKERS", "4")

		type Config struct {
			Port    int           `mapstructure:"PORT"`
			Rate    float64       `mapstructure:"RATE"`
			Debug   bool          `mapstructure:"DEBUG"`
			Timeout time.Duration `mapstructure:"TIMEOUT"`
			Hosts   []string      `mapstructure:"HOSTS"`
			Workers uint          `mapstructure:"WORKERS"`
		}

		var cfg Config
		err := LoadConfig(&cfg)
		assert.NoError(t, err)
		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, 1.5, cfg.Rate)
		assert.True(t, cfg.Debug)
		assert.Equal(t, 30*time.Second, cfg.Timeout)
		assert.Equal(t, []string{"a.com", "b.com", "c.com"}, cfg.Hosts)
		assert.Equal(t, uint(4), cfg.Workers)
	})

	t.Run("Defaults", func(t *testing.T) {
		os.Clearenv()
		type Config struct {
			Port int    `mapstructure:"PORT" default:"3000"`
			Env  string `mapstructure:"ENV" default:"development"`
		}
		var cfg Config
		err := LoadConfig(&cfg)
		assert.NoError(t, err)
		assert.Equal(t, 3000, cfg.Port)
		assert.Equal(t, "development", cfg.Env)
	})

	t.Run("RequiredMissing", func(t *testing.T) {
		os.Clearenv()
		type Config struct {
			Secret string `mapstructure:"SECRET" required:"true"`
		}
		var cfg Config
		err := LoadConfig(&cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SECRET")
	})

	t.Run("ParseError", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("PORT", "not-a-number")
		type Config struct {
			Port int `mapstructure:"PORT"`
		}
		var cfg Config
		err := LoadConfig(&cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PORT")
	})

	t.Run("OverflowRejected", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("SMALL", "200")
		type Config struct {
			Small int8 `mapstructure:"SMALL"`
		}
		var cfg Config
		err := LoadConfig(&cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SMALL")
	})

	t.Run("RejectsNonPointer", func(t *testing.T) {
		os.Clearenv()
		type Config struct {
			X string `mapstructure:"X"`
		}
		var cfg Config
		err := LoadConfig(cfg) // value, not pointer
		assert.Error(t, err)
	})

	t.Run("RejectsNil", func(t *testing.T) {
		os.Clearenv()
		err := LoadConfig(nil)
		assert.Error(t, err)
	})
}
