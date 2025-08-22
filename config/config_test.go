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

		config, err := LoadConfig(".")
		assert.NoError(t, err)
		assert.Equal(t, "test_secret", config.JWTSecret)
		assert.Equal(t, "debug", config.LogLevel)
	})
}
